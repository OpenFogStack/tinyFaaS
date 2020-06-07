import asyncio
import tornado.ioloop
import tornado.web
import zipfile, io
import base64
import docker
import json
import uuid
import shutil
import urllib
import sys
import shutil
import hashlib
CONFIG_PORT = 8080
endpoint_container = {}
function_handlers = {}

def create_endpoint(meta_container):
    client = docker.from_env()
    endpoint_image = client.images.build(path='./endpoint/', rm=True)[0]

    endpoint_network = client.networks.create('endpoint-net', driver='bridge')

    endpoint_container['container'] = client.containers.run(endpoint_image, network=endpoint_network.name, ports={'5683/tcp': 5683}, detach=True)
    # getting IP address of the handler container by inspecting the network and converting CIDR to IPv4 address notation (very dirtily, removing the last 3 chars -> i.e. '/20', so let's hope we don't have a /8 subnet mask)
    endpoint_container['ipaddr'] = docker.APIClient().inspect_network(endpoint_network.id)['Containers'][endpoint_container['container'].id]['IPv4Address'][:-3]

    endpoint_network.connect(meta_container)

class FunctionHandler():
    def __init__(self, function_name, function_resource, function_path, function_entry, function_threads, environment):
        self.client = docker.from_env()
        self.function_resource = function_resource
        self.name = function_name

        # copy all files in ./templates/functionhandler to handler-runtime/[function_path]
        shutil.copytree('./templates/functionhandler', './handler-runtime/' + self.name)

        # copy the folder ./handlers/[function_path] to handler-runtime/[function_path]
        shutil.copytree('./tmp', './handler-runtime/' + self.name + '/' + function_path)

        # use the Dockerfile.template to create a custom Dockerfile with function_path
        with open('./templates/Dockerfile.template', 'rt') as fin:
            with open('./handler-runtime/' + self.name + '/Dockerfile', 'wt') as fout:
                for line in fin:
                    fout.write(line.replace('%%%HANDLERPATH%%%', function_path))

        # use the functionhandler.js.template to create a custom functionhandler.js with function_path as a module name
        with open('./templates/functionhandler.js.template', 'rt') as fin:
            with open('./handler-runtime/' + self.name + '/functionhandler.js', 'wt') as fout:
                for line in fin:
                    fout.write(line.replace('%%%PACKAGENAME%%%', function_path))

        self.this_image = self.client.images.build(path='./handler-runtime/' + self.name, rm=True)[0]

        self.thread_count = function_threads

        # connect handler container(s) to endpoint on a dedicated subnet
        self.this_network = self.client.networks.create(self.name + '-net', driver='bridge')

        self.this_network.connect(endpoint_container['container'].name)

        self.this_handler_ips = list([None]*self.thread_count)

        # create handler container(s)
        self.this_containers = list([None]*self.thread_count)

        for i in range(0, self.thread_count):
            self.this_containers[i] = self.client.containers.run(self.this_image, environment=environment, network=self.this_network.name, detach=True)
            # getting IP address of the handler container by inspecting the network and converting CIDR to IPv4 address notation (very dirtily, removing the last 3 chars -> i.e. '/20', so let's hope we don't have a /8 subnet mask)
            self.this_handler_ips[i] = docker.APIClient().inspect_network(self.this_network.id)['Containers'][self.this_containers[i].id]['IPv4Address'][:-3]

        # tell endpoint about new function
        function_handler = {
            "function_resource": self.function_resource,
            "function_containers": self.this_handler_ips
        }

        data = json.dumps(function_handler).encode('ascii')

        urllib.request.urlopen(url='http://' + endpoint_container['ipaddr'] + ':80', data=data)
    def destroy(self):
        function_handler = {
            "function_resource": self.function_resource,
            "function_containers": []
        }
        data = json.dumps(function_handler).encode('ascii')
        urllib.request.urlopen(url='http://' + endpoint_container['ipaddr'] + ':80', data=data)


        for container in self.this_containers:
            container.remove(force=True)
        self.this_network.reload()
        for container in self.this_network.containers:
            self.this_network.disconnect(container, force=True)
        self.this_network.remove()

        shutil.rmtree('./handler-runtime/' + self.name)
        

class UploadHandler(tornado.web.RequestHandler):
    async def post(self):
        try:
            # expected post body
            # {
            #     name: "name"
            #     environment: {}
            #     threads: 2,
            #     zip: "base64-zip"
            # }
            #

            function_data = tornado.escape.json_decode(self.request.body)
            environment = function_data['environment']
            function_threads = function_data['threads']
            function_name = function_data['name'] + '-handler'
            function_path = function_data['name']

            function_resource = "/" + function_data['name']
            shutil.rmtree('./tmp', ignore_errors=True)
            function_zip = function_data['zip']   
            function_zip = base64.b64decode(function_zip)

            function_zip_file = io.BytesIO(function_zip)

            zip = zipfile.ZipFile(function_zip_file)
            zip.extractall(path='./tmp')

            package_json = tornado.escape.json_decode(open('./tmp/package.json').read())
            function_entry = package_json['main']
 

            if function_name in function_handlers:
                function_handlers[function_name].destroy()

            function_handlers[function_name] = FunctionHandler(function_name, function_resource, 
            function_path, function_entry, function_threads, environment)
            function_handlers[function_name].zip_hash = hashlib.sha256(function_zip).hexdigest()

        except Exception as e:
            raise
class DeleteHandler(tornado.web.RequestHandler):
    async def post(self):
        try:
            handler_name = self.request.body.decode("utf-8") + '-handler'
            # expected json {name: "function-name-from-pkg-json"}
            if handler_name in function_handlers:
                function_handlers[handler_name].destroy()
                del function_handlers[handler_name]
            else:
                self.write("Not found")

        except Exception as e:
            raise

class WipeHandler(tornado.web.RequestHandler):
    async def post(self):
        try:
            for f in function_handlers:
                function_handlers[f].destroy()
            function_handlers.clear()
        except Exception as e:
            raise


class ListHandler(tornado.web.RequestHandler):
    async def get(self):
        try:
            out = []
            for f in function_handlers:
                out.append({"name": function_handlers[f].name[:-len("-handler")], 
                "hash": function_handlers[f].zip_hash, 
                "threads": function_handlers[f].thread_count,
                "resource": function_handlers[f].function_resource})
            self.write(json.dumps(out) + '\n')
        except Exception as e:
            raise


def main(args):
    # read config data
    # exactly one argument should be provided: meta_container
    if len(args) != 2:
        raise ValueError('Too many or too little arguments provided:\n' + json.dumps(args))

    meta_container = args[1]

    try:
      docker.from_env().containers.get(meta_container)
    except:
      raise ValueError('Provided container name does not match a running container')

    # create endpoint
    create_endpoint(meta_container)

    # accept incoming configuration requests and create handlers based on that
    app = tornado.web.Application([
        (r'/upload', UploadHandler),
        (r'/delete', DeleteHandler),
        (r'/list', ListHandler),
        (r'/wipe', WipeHandler)
    ])
    app.listen(CONFIG_PORT)
    tornado.ioloop.IOLoop.current().start()

if __name__ == '__main__':
    main(sys.argv)
