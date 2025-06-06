const express = require('express');
const jwt = require('jsonwebtoken');
const nodemailer = require('nodemailer');
const User = require('../models/User');
const { protect } = require('../middleware/auth');

const router = express.Router();

function signToken(id) {
  return jwt.sign({ id }, process.env.JWT_SECRET, { expiresIn: '1d' });
}

router.post('/register', async (req, res) => {
  try {
    const user = await User.create(req.body);
    const token = signToken(user._id);
    res.json({ token, user });
  } catch(err) {
    res.status(400).json({ message: err.message });
  }
});

router.post('/login', async (req, res) => {
  const { email, password } = req.body;
  const user = await User.findOne({ email });
  if (!user || !(await user.comparePassword(password))) {
    return res.status(401).json({ message: 'Invalid credentials' });
  }
  const token = signToken(user._id);
  res.json({ token, user });
});

router.post('/forgot', async (req, res) => {
  const { email } = req.body;
  const user = await User.findOne({ email });
  if (!user) return res.status(404).json({ message: 'No user' });
  const token = signToken(user._id);
  user.resetToken = token;
  user.resetTokenExp = Date.now() + 3600000;
  await user.save();

  const transporter = nodemailer.createTransport({
    service: 'gmail',
    auth: {
      user: process.env.EMAIL_USER,
      pass: process.env.EMAIL_PASS
    }
  });

  const resetUrl = `${process.env.CLIENT_URL}/reset/${token}`;
  await transporter.sendMail({
    to: user.email,
    subject: 'Password Reset',
    html: `<p>Reset your password <a href="${resetUrl}">here</a></p>`
  });

  res.json({ message: 'Email sent' });
});

router.post('/reset/:token', async (req, res) => {
  const { token } = req.params;
  const user = await User.findOne({ resetToken: token, resetTokenExp: { $gt: Date.now() } });
  if (!user) return res.status(400).json({ message: 'Invalid or expired token' });
  user.password = req.body.password;
  user.resetToken = undefined;
  user.resetTokenExp = undefined;
  await user.save();
  res.json({ message: 'Password updated' });
});

router.get('/me', protect, (req, res) => {
  res.json(req.user);
});

module.exports = router;
