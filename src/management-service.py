import asyncio
import tornado.ioloop
import tornado.web

import docker
import json
import uuid
import shutil
import urllib
import sys

CONFIG_PORT = 8080
endpoint_container = {}
function_handlers = {}

def create_endpoint(meta_container, port):
    client = docker.from_env()
    endpoint_image = client.images.build(path='./reverse-proxy/', rm=True)[0]

    endpoint_network = client.networks.create('endpoint-net', driver='bridge')

    endpoint_container['container'] = client.containers.run(endpoint_image, network=endpoint_network.name, ports={'5683/udp': port}, detach=True)
    # getting IP address of the handler container by inspecting the network and converting CIDR to IPv4 address notation (very dirtily, removing the last 3 chars -> i.e. '/20', so let's hope we don't have a /8 subnet mask)
    endpoint_container['ipaddr'] = docker.APIClient().inspect_network(endpoint_network.id)['Containers'][endpoint_container['container'].id]['IPv4Address'][:-3]

    endpoint_network.connect(meta_container)

class FunctionHandler():
    def __init__(self, function_name, function_resource, function_path, function_entry, function_threads):
        self.client = docker.from_env()
        self.function_resource = function_resource
        self.name = function_name

        # copy all files in ./templates/functionhandler to handler-runtime/[function_path]
        shutil.copytree('./templates/functionhandler', './handler-runtime/' + self.name)

        # copy the folder ./handlers/[function_path] to handler-runtime/[function_path]
        shutil.copytree('./handlers/' + function_path, './handler-runtime/' + self.name + '/' + function_path)

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
            self.this_containers[i] = self.client.containers.run(self.this_image, network=self.this_network.name, detach=True)
            # getting IP address of the handler container by inspecting the network and converting CIDR to IPv4 address notation (very dirtily, removing the last 3 chars -> i.e. '/20', so let's hope we don't have a /8 subnet mask)
            self.this_handler_ips[i] = docker.APIClient().inspect_network(self.this_network.id)['Containers'][self.this_containers[i].id]['IPv4Address'][:-3]

        # tell endpoint about new function
        function_handler = {
            "function_resource": self.function_resource,
            "function_containers": self.this_handler_ips
        }

        data = json.dumps(function_handler).encode('ascii')

        urllib.request.urlopen(url='http://' + endpoint_container['ipaddr'] + ':80', data=data)

class EndpointHandler(tornado.web.RequestHandler):
    async def post(self):
        try:
            # expected post body
            # {
            #     path: 'handler-path',
            #     resource: 'han/dler',
            #     entry: 'handler.js',
            #     threads: 2
            # }
            #

            function_data = tornado.escape.json_decode(self.request.body)

            function_path = function_data['path']
            function_resource = function_data['resource']
            function_entry = function_data['entry']
            function_threads = function_data['threads']

            function_name = str(uuid.uuid4()) + '-' + function_path + '-handler'

            function_handlers[function_name] = FunctionHandler(function_name, function_resource, function_path, function_entry, function_threads)


        except Exception as e:
            raise

def main(args):
    
    # default coap port is 5683
    port = 5683

    if len(args) == 3:
        try:
            port = int(args[2])
        except ValueError:
                    raise ValueError('Could not parse port number:\n' + json.dumps(args) + '\nUsage: management-service.py [tinyfaas-mgmt container name] <endpoint port>')
    elif len(args) != 2:
        raise ValueError('Too many or too little arguments provided:\n' + json.dumps(args) + '\nUsage: management-service.py [tinyfaas-mgmt container name] <endpoint port>')

    meta_container = args[1]

    try:
      docker.from_env().containers.get(meta_container)
    except:
      raise ValueError('Provided container name does not match a running container' + '\nUsage: management-service.py [tinyfaas-mgmt container name] <endpoint port>')

    # create endpoint
    create_endpoint(meta_container, port)

    # accept incoming configuration requests and create handlers based on that
    app = tornado.web.Application([
        (r'/', EndpointHandler),
    ])
    app.listen(CONFIG_PORT)
    tornado.ioloop.IOLoop.current().start()

if __name__ == '__main__':
    main(sys.argv)
