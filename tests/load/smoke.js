import http from "k6/http";
import { check, sleep } from "k6";

export const options = { vus: 5, duration: "30s" };

export default function () {
    let res = http.get(__ENV.BASE_URL + "/healthz");
    check(res, { "health 204": (r) => r.status === 204 });

    const email = `t${__VU}@ex.com`;
    http.post(__ENV.BASE_URL + "/auth/register", JSON.stringify({ email, password: "p@ss" }), { headers: { "Content-Type": "application/json" } });
    const login = http.post(__ENV.BASE_URL + "/auth/login", JSON.stringify({ email, password: "p@ss" }), { headers: { "Content-Type": "application/json" } });
    const token = login.json("access");
    const note = http.post(__ENV.BASE_URL + "/notes", JSON.stringify({ title: "hi", body: "there" }), { headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` } });
    check(note, { "note ok": (r) => r.status === 200 });

    sleep(1);
}
