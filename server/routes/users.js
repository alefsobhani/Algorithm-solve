const express = require('express');
const User = require('../models/User');
const { protect, admin } = require('../middleware/auth');

const router = express.Router();

router.get('/', protect, admin, async (req, res) => {
  const users = await User.find().select('-password');
  res.json(users);
});

router.put('/:id/role', protect, admin, async (req, res) => {
  const user = await User.findById(req.params.id);
  if (!user) return res.status(404).json({ message: 'User not found' });
  user.role = req.body.role;
  await user.save();
  res.json(user);
});

module.exports = router;
