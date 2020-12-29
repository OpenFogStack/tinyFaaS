'use strict';
process.chdir("fn");

const handler = require('fn')
const express = require('express')
const app = express()

app.all('/*', handler);
app.listen(8000)
