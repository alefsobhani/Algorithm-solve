const express = require('express');
const Task = require('../models/Task');
const Comment = require('../models/Comment');
const { protect } = require('../middleware/auth');

const router = express.Router();

router.get('/', protect, async (req, res) => {
  const query = { $or: [ { createdBy: req.user._id }, { assignedTo: req.user._id } ] };
  const tasks = await Task.find(query).populate('assignedTo', 'name');
  res.json(tasks);
});

router.post('/', protect, async (req, res) => {
  const task = await Task.create({ ...req.body, createdBy: req.user._id });
  res.json(task);
});

router.put('/:id', protect, async (req, res) => {
  const task = await Task.findById(req.params.id);
  if (!task) return res.status(404).json({ message: 'Task not found' });
  if (task.createdBy.toString() !== req.user._id.toString() && req.user.role !== 'admin') {
    return res.status(403).json({ message: 'Not authorized' });
  }
  Object.assign(task, req.body);
  await task.save();
  res.json(task);
});

router.delete('/:id', protect, async (req, res) => {
  const task = await Task.findById(req.params.id);
  if (!task) return res.status(404).json({ message: 'Task not found' });
  if (task.createdBy.toString() !== req.user._id.toString() && req.user.role !== 'admin') {
    return res.status(403).json({ message: 'Not authorized' });
  }
  await task.remove();
  res.json({ message: 'Task removed' });
});

// comments
router.post('/:id/comments', protect, async (req, res) => {
  const task = await Task.findById(req.params.id);
  if (!task) return res.status(404).json({ message: 'Task not found' });
  const comment = await Comment.create({
    text: req.body.text,
    author: req.user._id,
    task: task._id,
    taggedUsers: req.body.taggedUsers
  });
  task.comments.push(comment._id);
  await task.save();
  res.json(comment);
});

module.exports = router;
