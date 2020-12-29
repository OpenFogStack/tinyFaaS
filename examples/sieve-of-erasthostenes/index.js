
module.exports = (req, res) => {
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

  res.send(primes.toString() + "\n");

}
