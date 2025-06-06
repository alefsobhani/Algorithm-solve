const express = require('express');
const Project = require('../models/Project');
const { protect } = require('../middleware/auth');

const router = express.Router();

router.get('/', protect, async (req, res) => {
  const projects = await Project.find().populate('manager', 'name');
  res.json(projects);
});

router.post('/', protect, async (req, res) => {
  const project = await Project.create({ ...req.body, manager: req.user._id });
  res.json(project);
});

router.put('/:id', protect, async (req, res) => {
  const project = await Project.findByIdAndUpdate(req.params.id, req.body, { new: true });
  res.json(project);
});

router.delete('/:id', protect, async (req, res) => {
  await Project.findByIdAndDelete(req.params.id);
  res.json({ message: 'Project removed' });
});

module.exports = router;
