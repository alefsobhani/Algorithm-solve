const request = require('supertest');
const express = require('express');
const authRoutes = require('../routes/auth');

const app = express();
app.use(express.json());
app.use('/api/auth', authRoutes);

test('GET /api/auth/me requires token', async () => {
  const res = await request(app).get('/api/auth/me');
  expect(res.status).toBe(401);
});
