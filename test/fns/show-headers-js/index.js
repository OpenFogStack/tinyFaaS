
module.exports = (req, res) => {
  const body = req.body;
  const headers = req.headers; // headers from the http request or GRPC metadata

  console.log("Headers:", headers);

  res.send(headers);
}
