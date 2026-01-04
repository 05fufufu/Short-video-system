import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
  vus: 50, // 50 个并发用户
  duration: '30s', // 持续 30 秒
};

export default function () {
  // 1. 访问 Feed 接口
  let res = http.get('http://localhost:8080/feed');
  
  // 2. 检查响应状态是否为 200
  check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });

  sleep(1); // 每个用户请求后休息 1 秒
}
