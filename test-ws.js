const WebSocket = require('ws');
const token = process.env.TOKEN;
const ws = new WebSocket(`ws://localhost:8080/ws/sandbox/${process.env.SANDBOX_ID}`, {
  headers: {
    'Origin': 'http://localhost:3000'
  }
});

ws.on('open', function open() {
  console.log('connected');
  ws.send('ls -la\r');
  setTimeout(() => ws.close(), 2000);
});

ws.on('message', function incoming(data) {
  console.log(data.toString());
});

ws.on('error', function error(err) {
  console.error('error', err);
});
