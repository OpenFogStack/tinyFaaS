module.exports = {
  eventhandler: function (req, res) {
    console.log(req.url)
    const max = 1000;
    let sieve = [], i, j, primes = [];
    for (i = 2; i <= max; ++i) {

        if (!sieve[i]) {
            primes.push(i);
            for (j = i << 1; j <= max; j += i) {
                sieve[j] = true;
            }
        }
    }
    console.log("Found primes: " + primes.toString());


    res.end(JSON.stringify({
      response_code: "2.05",
      payload: primes.toString()
    }));
    
  }
}
