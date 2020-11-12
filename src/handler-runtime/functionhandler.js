'use strict';
process.chdir("fn");

const handler = require('fn')
const express = require('express')
const app = express()
const port = 8000


app.all('/*', handler.tinyfaasHandler);
app.listen(port, () => console.log(`Example app listening at http://localhost:${port}`))
