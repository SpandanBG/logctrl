const sleep_time = 200

async function waitFor(ms) {
  return new Promise(res => setTimeout(res, ms))
}

async function makeLogs() {
  console.log("hello")
  await waitFor(sleep_time)
  for (let i = 0; i < 10_000; i++) {
    console.log(i % 2 == 0 ? 'even' : 'odd', ' ', i)
    await waitFor(sleep_time)
  }
  console.log("world")
}

makeLogs()
