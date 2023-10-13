"use strict";
process.chdir("fn");

const handler = require("fn");
const express = require("express");
const bodyParser = require("body-parser")
const app = express();

app.use(bodyParser.text({
    type: function(req) {
        return 'text';
    }
}));

app.all("/health", (req, res) => {
  return res.send("OK");
});
app.all("/fn", handler);
app.listen(8000);
