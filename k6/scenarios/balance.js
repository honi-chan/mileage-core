import http from 'k6/http';
import { check, sleep } from 'k6';

// 設定
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
let TOKEN = '';

export const options = {
    scenarios: {
        balance_read: {
            executor: 'constant-arrival-rate',
            rate: 200,
            timeUnit: '1s',
            duration: '30s',
            preAllocatedVUs: 50,
            maxVUs: 100,
        },
    },
    thresholds: {
        'http_req_duration{scenario:balance_read}': ['p(99)<50'],
    },
};

export function setup() {
    // テストユーザー作成
    const signupRes = http.post(`${BASE_URL}/v1/auth/signup`, JSON.stringify({
        email: `k6-balance-${Date.now()}@test.com`,
        password: 'testpass123',
    }), { headers: { 'Content-Type': 'application/json' } });

    const body = JSON.parse(signupRes.body);

    // 初期マイル付与
    http.post(`${BASE_URL}/v1/mileage/grant`, JSON.stringify({
        amount: 10000,
        reason: 'k6_setup',
    }), {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${body.token}`,
            'Idempotency-Key': `k6-setup-${Date.now()}`,
        },
    });

    return { token: body.token };
}

export default function (data) {
    const res = http.get(`${BASE_URL}/v1/mileage/balance`, {
        headers: {
            'Authorization': `Bearer ${data.token}`,
        },
    });

    check(res, {
        'status is 200': (r) => r.status === 200,
        'has balance': (r) => JSON.parse(r.body).balance !== undefined,
    });
}
