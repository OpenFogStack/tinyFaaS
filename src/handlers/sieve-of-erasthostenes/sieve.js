module.exports = {
  eventhandler: function () {
    const max = 1000;
    let sieve = [], i, j, primes = [];
    for (i = 2; i <= max; ++i) {

        if (!sieve[i]) {
            console.log("Found prime: " + i);
            primes.push(i);
            for (j = i << 1; j <= max; j += i) {
                sieve[j] = true;
            }
        }
    }

    return JSON.stringify({
      response_code: "2.05",
      payload: primes.toString()
    });
  }
}
