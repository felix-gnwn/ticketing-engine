import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

http.setResponseCallback(http.expectedStatuses(201, 409));

// k6 Load Testing Options & SLA Thresholds
export const options = {
    stages: [
        { duration: '10s', target: 10 }, // Ramp-up: 10 VUs
        { duration: '30s', target: 100 }, // Peak Stress Test: 100 VUs (Flash Sale Scenario)
        { duration: '10s', target: 0 }, // Ramp-down: 0 VUs
    ],
    thresholds: {
        http_req_failed: ['rate<0.01'], // Error rate must be under 1%
        http_req_duration: ['p(95)<500', 'p(99)<1000'], // SLA: 95% < 500ms, 99% < 1000ms
    },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Setup Phase: Automated login to obtain JWT Bearer Token before stress testing begins
export function setup() {
    const loginPayload = JSON.stringify({
        email: 'test@example.com',
        password: 'secret123',
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    const res = http.post(`${BASE_URL}/api/v1/auth/login`, loginPayload, params);

    if (res.status === 200) {
        const body = JSON.parse(res.body);
        
        return {
            token: body.data.token
        };
    }

    console.log(`[SETUP FAILED] Unable to authenticate test user: ${res.status} - ${res.body}`);
    return {
        token: ''
    };
}

// Default VU Iteration Logic
export default function (data) {
    const idempotencyKey = `k6-checkout-${Date.now()}-${randomString(8)}`;

    const checkoutPayload = JSON.stringify({
        ticket_id: 'evt-1794632310975768181',
        quantity: 1,
    });

    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${data.token}`,
            'X-Idempotency-Key': idempotencyKey,
        },
    };

    const res = http.post(`${BASE_URL}/api/v1/orders`, checkoutPayload, params);

    // Assert response status: 201 (Created) or 409 (Conflict / Out of Stock) are valid business responses
    check(res, {
        'status is 201 or 409': (r) => r.status === 201 || r.status === 409,
        'response time < 500ms': (r) => r.timings.duration < 500,
    });

    sleep(0.1); // Short think-time between request iterations
}