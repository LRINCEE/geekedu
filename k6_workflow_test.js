import http from 'k6/http';
import { check, group, sleep } from 'k6';

export const options = {
  stages: [
    { duration: __ENV.RAMP_UP || '10s', target: Number(__ENV.VUS || 20) },
    { duration: __ENV.HOLD || '30s', target: Number(__ENV.VUS || 20) },
    { duration: __ENV.RAMP_DOWN || '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<500'],
    checks: ['rate>0.95'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1';
const PASSWORD = __ENV.PASSWORD || 'password123';
const FIXED_USERNAME = __ENV.USERNAME || '';
const FIXED_COURSE_ID = Number(__ENV.COURSE_ID || 0);
const FIXED_VIDEO_ID = Number(__ENV.VIDEO_ID || 0);
const SKIP_REGISTER = (__ENV.SKIP_REGISTER || (FIXED_USERNAME ? 'true' : 'false')) === 'true';
const ALLOW_ALREADY_PURCHASED = (__ENV.ALLOW_ALREADY_PURCHASED || 'true') === 'true';
const REQUIRE_VIDEO = (__ENV.REQUIRE_VIDEO || 'true') === 'true';
const THINK_TIME = Number(__ENV.THINK_TIME || 1);

function json(res, path, fallback = undefined) {
  try {
    const value = path ? res.json(path) : res.json();
    return value === undefined ? fallback : value;
  } catch (_) {
    return fallback;
  }
}

function authHeaders(token) {
  return {
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
    },
  };
}

function publicJsonHeaders() {
  return { headers: { 'Content-Type': 'application/json' } };
}

function uniqueUsername() {
  const suffix = Math.random().toString(36).slice(2, 10);
  return `k6user_${__VU}_${__ITER}_${suffix}`;
}

export default function () {
  const username = FIXED_USERNAME || uniqueUsername();
  let token = '';
  let courseId = FIXED_COURSE_ID;
  let videoId = FIXED_VIDEO_ID;

  if (!SKIP_REGISTER) {
    group('01_register', () => {
      const payload = JSON.stringify({ username, password: PASSWORD });
      const res = http.post(`${BASE_URL}/auth/register`, payload, publicJsonHeaders());

      check(res, {
        'register http 200': (r) => r.status === 200,
        'register code 0': (r) => json(r, 'code') === 0,
        'register returns user_id': (r) => Number(json(r, 'data.user_id', 0)) > 0,
      });
    });

    sleep(THINK_TIME);
  }

  group('02_login', () => {
    const payload = JSON.stringify({ username, password: PASSWORD });
    const res = http.post(`${BASE_URL}/auth/login`, payload, publicJsonHeaders());

    check(res, {
      'login http 200': (r) => r.status === 200,
      'login code 0': (r) => json(r, 'code') === 0,
      'login returns token': (r) => typeof json(r, 'data.token', '') === 'string' && json(r, 'data.token', '').length > 0,
    });

    token = json(res, 'data.token', '');
  });

  if (!token) {
    return;
  }

  sleep(THINK_TIME);

  group('03_list_courses', () => {
    const res = http.get(`${BASE_URL}/courses?page=1&page_size=10`, authHeaders(token));
    const courses = json(res, 'data.courses', []);

    check(res, {
      'list courses http 200': (r) => r.status === 200,
      'list courses code 0': (r) => json(r, 'code') === 0,
      'list courses non-empty': () => Array.isArray(courses) && courses.length > 0,
    });

    if (!courseId && Array.isArray(courses) && courses.length > 0) {
      courseId = Number(courses[0].id || 0);
    }
  });

  if (!courseId) {
    check(null, { 'course id available': () => false });
    return;
  }

  sleep(THINK_TIME);

  group('04_create_order', () => {
    const payload = JSON.stringify({ course_id: courseId });
    const res = http.post(`${BASE_URL}/orders`, payload, authHeaders(token));
    const code = json(res, 'code');

    check(res, {
      'create order success or already purchased': (r) => {
        if (r.status === 200 && code === 0) return true;
        return ALLOW_ALREADY_PURCHASED && r.status === 409 && code === 40001;
      },
    });
  });

  sleep(THINK_TIME);

  group('05_get_course_detail', () => {
    const res = http.get(`${BASE_URL}/courses/${courseId}`, authHeaders(token));
    const videos = json(res, 'data.videos', []);

    check(res, {
      'course detail http 200': (r) => r.status === 200,
      'course detail code 0': (r) => json(r, 'code') === 0,
    });

    if (!videoId && Array.isArray(videos) && videos.length > 0) {
      videoId = Number(videos[0].id || 0);
    }
  });

  if (!videoId) {
    check(null, { 'video id available': () => !REQUIRE_VIDEO });
    return;
  }

  sleep(THINK_TIME);

  group('06_get_play_url', () => {
    const res = http.get(`${BASE_URL}/player/${videoId}`, authHeaders(token));

    check(res, {
      'play url http 200': (r) => r.status === 200,
      'play url code 0': (r) => json(r, 'code') === 0,
      'play url returned': (r) => typeof json(r, 'data.play_url', '') === 'string' && json(r, 'data.play_url', '').length > 0,
    });
  });

  sleep(THINK_TIME);
}
