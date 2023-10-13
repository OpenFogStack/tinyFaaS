
module.exports = (req, res) => {
  const response = req.body;

  console.log(response);

  res.send(response); 
}
