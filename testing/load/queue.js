import http from 'k6/http';
import { check } from 'k6';

const queue_name = "autmations";
const queue_type = "delayed";

export function setup() {
  const url = `http://localhost:8001/API/v1/queues`;
  let payload = JSON.stringify(
    {"name": queue_name, "type": queue_type}
  );

  let params = {
    headers: {
      "Content-Type": "application/json",
      "Accept": "application/json, application/problem+json"
    },
    tags: {},
  };

  let resp = http.post(url, payload, params);
}

export function teardown(data) {

  const url = `http://localhost:8001/API/v1/queues/${queue_name}`;

  let params = {
    headers: {
      "Accept": "application/json, application/problem+json"
    },
    tags: {},
  };

  let resp = http.del(url, null, params);
}

export default function () {
  const url = `http://localhost:8001/API/v1/queues/${queue_name}/messages`;
  let payload = JSON.stringify(
    {"content": "{\"user_id\": 10}", "priority": 10, "group": "default"}
  );

  let params = {
    headers: {
      "Content-Type": "application/json",
      "Accept": "application/json, application/problem+json"
    },
    tags: { name: 'enqueue' },
  };

  let resp = http.post(url, payload, params);

  check(resp, {
    'enqueued': (r) => r.status === 200,
  });

  params = {
    headers: {
      "Accept": "application/json, application/problem+json"
    },
    tags: { name: 'dequeue' },
  };

  resp = http.request("GET", url + '?ack=true', {}, params);
  check(resp, {
    'dequeued': (r) => r.status === 200,
  });
}
