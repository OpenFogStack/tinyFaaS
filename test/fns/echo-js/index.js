
module.exports = (req, res) => {
  const response = req.body;
  const headers = req.headers; // headers from the http request or GRPC metadata

  console.log("Headers:", headers);
  console.log(response);

  res.send(response); 
}
