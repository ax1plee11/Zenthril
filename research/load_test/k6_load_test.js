// Нагрузочный тест для Zenthril с использованием k6
// Установка: https://k6.io/docs/getting-started/installation/
// Запуск: k6 run k6_load_test.js

import http from 'k6/http';
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

// Кастомные метрики
const messageLatency = new Trend('message_latency');
const wsConnections = new Counter('ws_connections');
const messagesSent = new Counter('messages_sent');
const messagesReceived = new Counter('messages_received');

// Конфигурация теста
export const options = {
  stages: [
    { duration: '30s', target: 50 },   // Разогрев: 50 пользователей
    { duration: '1m', target: 100 },   // Рост до 100 пользователей
    { duration: '2m', target: 500 },   // Рост до 500 пользователей
    { duration: '3m', target: 1000 },  // Пик: 1000 пользователей
    { duration: '2m', target: 500 },   // Снижение до 500
    { duration: '1m', target: 100 },   // Снижение до 100
    { duration: '30s', target: 0 },    // Остановка
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% запросов < 500ms, 99% < 1s
    http_req_failed: ['rate<0.01'],                  // Менее 1% ошибок
    message_latency: ['p(95)<200', 'p(99)<500'],    // Задержка сообщений
    ws_connections: ['count>0'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const WS_URL = __ENV.WS_URL || 'ws://localhost:8080';

// Генерация случайного имени пользователя
function randomUsername() {
  return `user_${Math.random().toString(36).substring(7)}_${Date.now()}`;
}

// Регистрация пользователя
function register() {
  const username = randomUsername();
  const password = 'TestPassword123!';
  
  const payload = JSON.stringify({
    username: username,
    password: password,
  });
  
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };
  
  const res = http.post(`${BASE_URL}/api/v1/auth/register`, payload, params);
  
  check(res, {
    'registration successful': (r) => r.status === 200 || r.status === 201,
  });
  
  return { username, password };
}

// Вход пользователя
function login(username, password) {
  const payload = JSON.stringify({
    username: username,
    password: password,
  });
  
  const params = {
    headers: { 'Content-Type': 'application/json' },
  };
  
  const res = http.post(`${BASE_URL}/api/v1/auth/login`, payload, params);
  
  check(res, {
    'login successful': (r) => r.status === 200,
    'token received': (r) => r.json('token') !== undefined,
  });
  
  if (res.status === 200) {
    return res.json('token');
  }
  return null;
}

// Получение WebSocket ticket
function getWSTicket(token) {
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  };
  
  const res = http.post(`${BASE_URL}/api/v1/auth/ws-ticket`, null, params);
  
  check(res, {
    'ws ticket received': (r) => r.status === 200,
  });
  
  if (res.status === 200) {
    return res.json('ticket');
  }
  return null;
}

// Создание гильдии
function createGuild(token) {
  const payload = JSON.stringify({
    name: `Test Guild ${Date.now()}`,
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  };
  
  const res = http.post(`${BASE_URL}/api/v1/guilds`, payload, params);
  
  check(res, {
    'guild created': (r) => r.status === 200 || r.status === 201,
  });
  
  if (res.status === 200 || res.status === 201) {
    return res.json('id');
  }
  return null;
}

// Создание канала
function createChannel(token, guildId) {
  const payload = JSON.stringify({
    name: `test-channel-${Date.now()}`,
    type: 'text',
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  };
  
  const res = http.post(`${BASE_URL}/api/v1/guilds/${guildId}/channels`, payload, params);
  
  check(res, {
    'channel created': (r) => r.status === 200 || r.status === 201,
  });
  
  if (res.status === 200 || res.status === 201) {
    return res.json('id');
  }
  return null;
}

// Отправка сообщения через HTTP
function sendMessage(token, channelId, content) {
  const payload = JSON.stringify({
    content: content,
  });
  
  const params = {
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
    },
  };
  
  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/api/v1/channels/${channelId}/messages`, payload, params);
  const latency = Date.now() - startTime;
  
  check(res, {
    'message sent': (r) => r.status === 200 || r.status === 201,
  });
  
  messageLatency.add(latency);
  messagesSent.add(1);
  
  return res.status === 200 || res.status === 201;
}

// Основной сценарий теста
export default function () {
  // 1. Регистрация
  const user = register();
  sleep(1);
  
  // 2. Вход
  const token = login(user.username, user.password);
  if (!token) {
    console.error('Login failed');
    return;
  }
  sleep(1);
  
  // 3. Создание гильдии
  const guildId = createGuild(token);
  if (!guildId) {
    console.error('Guild creation failed');
    return;
  }
  sleep(1);
  
  // 4. Создание канала
  const channelId = createChannel(token, guildId);
  if (!channelId) {
    console.error('Channel creation failed');
    return;
  }
  sleep(1);
  
  // 5. Отправка нескольких сообщений
  for (let i = 0; i < 5; i++) {
    sendMessage(token, channelId, `Test message ${i} from ${user.username}`);
    sleep(0.5);
  }
  
  // 6. WebSocket тест
  const wsTicket = getWSTicket(token);
  if (wsTicket) {
    const url = `${WS_URL}/ws?ticket=${wsTicket}`;
    
    const res = ws.connect(url, {}, function (socket) {
      wsConnections.add(1);
      
      socket.on('open', () => {
        console.log('WebSocket connected');
        
        // Отправляем несколько сообщений через WebSocket
        for (let i = 0; i < 3; i++) {
          const msg = JSON.stringify({
            type: 'message',
            channel_id: channelId,
            content: `WS message ${i}`,
          });
          
          const startTime = Date.now();
          socket.send(msg);
          messagesSent.add(1);
        }
      });
      
      socket.on('message', (data) => {
        messagesReceived.add(1);
        console.log('Message received:', data);
      });
      
      socket.on('error', (e) => {
        console.error('WebSocket error:', e);
      });
      
      // Держим соединение открытым 10 секунд
      sleep(10);
      socket.close();
    });
    
    check(res, {
      'ws connection successful': (r) => r && r.status === 101,
    });
  }
  
  sleep(2);
}

// Функция для отображения результатов
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'research/load_test/results.json': JSON.stringify(data, null, 2),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;
  
  let summary = '\n';
  summary += `${indent}Test Summary:\n`;
  summary += `${indent}  Duration: ${data.state.testRunDurationMs}ms\n`;
  summary += `${indent}  VUs: ${data.metrics.vus.values.max}\n`;
  summary += `${indent}  Iterations: ${data.metrics.iterations.values.count}\n`;
  summary += `${indent}\n`;
  summary += `${indent}HTTP Metrics:\n`;
  summary += `${indent}  Requests: ${data.metrics.http_reqs.values.count}\n`;
  summary += `${indent}  Failed: ${data.metrics.http_req_failed.values.rate * 100}%\n`;
  summary += `${indent}  Duration (avg): ${data.metrics.http_req_duration.values.avg}ms\n`;
  summary += `${indent}  Duration (p95): ${data.metrics.http_req_duration.values['p(95)']}ms\n`;
  summary += `${indent}  Duration (p99): ${data.metrics.http_req_duration.values['p(99)']}ms\n`;
  
  return summary;
}
