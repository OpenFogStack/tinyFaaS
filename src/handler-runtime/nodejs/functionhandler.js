'use strict';
process.chdir("fn");

const handler = require('fn')
const express = require('express')
const app = express()

app.all('/health', (req, res) => {return res.send('OK')})
app.all('/fn', handler);
app.listen(8000)
