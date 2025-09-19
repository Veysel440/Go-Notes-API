import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = { vus: 10, duration: '1m' };

const BASE = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
    check(http.get(`${BASE}/healthz`), { '204': r => r.status === 204 });

    const email = `u${__ITER % 1000}@t.io`;
    let res = http.post(`${BASE}/auth/register`, JSON.stringify({email, password:'Password1!'}), { headers: { 'Content-Type':'application/json' }});
    if (res.status !== 200 && res.status !== 409) { return; }

    res = http.post(`${BASE}/auth/login`, JSON.stringify({email, password:'Password1!'}), { headers: { 'Content-Type':'application/json' }});
    check(res, { 'login 200': r => r.status === 200 });
    const tok = res.json('access');
    const headers = { Authorization: `Bearer ${tok}`, 'Content-Type':'application/json' };

    res = http.post(`${BASE}/notes`, JSON.stringify({title:'t', body:'b'}), { headers });
    check(res, { 'note create 200': r => r.status === 200 });
    const id = res.json('id') || (res.json('id') === undefined ? res.json('id') : res.json('id')); // tolerant

    const g1 = http.get(`${BASE}/notes/${id}`, { headers });
    const etag = g1.headers['ETag'];
    const g2 = http.get(`${BASE}/notes/${id}`, { headers: { ...headers, 'If-None-Match': etag }});
    check(g2, { '304 on If-None-Match': r => r.status === 304 });

    sleep(1);
}
