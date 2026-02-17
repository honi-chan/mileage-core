import http from 'k6/http';
import { check } from 'k6';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

export const options = {
    scenarios: {
        contention: {
            executor: 'per-vu-iterations',
            vus: 10,
            iterations: 5,
            maxDuration: '30s',
        },
    },
    thresholds: {
        'http_req_duration': ['p(99)<500'],
        'checks': ['rate>0.8'],
    },
};

export function setup() {
    // テストユーザー作成
    const signupRes = http.post(`${BASE_URL}/v1/auth/signup`, JSON.stringify({
        email: `k6-contention-${Date.now()}@test.com`,
        password: 'testpass123',
    }), { headers: { 'Content-Type': 'application/json' } });

    const body = JSON.parse(signupRes.body);
    const token = body.token;

    // 十分なマイルを付与
    http.post(`${BASE_URL}/v1/mileage/grant`, JSON.stringify({
        amount: 100000,
        reason: 'k6_contention_setup',
    }), {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${token}`,
            'Idempotency-Key': `k6-contention-setup-${Date.now()}`,
        },
    });

    return { token };
}

export default function (data) {
    // 全VUが同一ユーザーのredeemを同時に実行 → 競合テスト
    const idempotencyKey = `contention-${__VU}-${__ITER}-${Date.now()}`;

    const res = http.post(`${BASE_URL}/v1/mileage/redeem`, JSON.stringify({
        amount: 10,
        reason: 'k6_contention_test',
    }), {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${data.token}`,
            'Idempotency-Key': idempotencyKey,
        },
    });

    check(res, {
        'status is 200 or 409': (r) => r.status === 200 || r.status === 409,
        'no 500 error': (r) => r.status !== 500,
    });
}
