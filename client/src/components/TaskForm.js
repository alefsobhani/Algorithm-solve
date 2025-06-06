import React, { useState } from 'react';
import api from '../services/api';

const TaskForm = ({ refresh }) => {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [dueDate, setDueDate] = useState('');
  const [priority, setPriority] = useState('low');
  const [status, setStatus] = useState('todo');

  const handleSubmit = async e => {
    e.preventDefault();
    await api.post('/tasks', { title, description, dueDate, priority, status });
    setTitle('');
    setDescription('');
    setDueDate('');
    setPriority('low');
    setStatus('todo');
    refresh();
  };

  return (
    <form onSubmit={handleSubmit}>
      <input placeholder="Title" value={title} onChange={e => setTitle(e.target.value)} />
      <input placeholder="Description" value={description} onChange={e => setDescription(e.target.value)} />
      <input type="date" value={dueDate} onChange={e => setDueDate(e.target.value)} />
      <select value={priority} onChange={e => setPriority(e.target.value)}>
        <option value="low">Low</option>
        <option value="medium">Medium</option>
        <option value="high">High</option>
      </select>
      <select value={status} onChange={e => setStatus(e.target.value)}>
        <option value="todo">To Do</option>
        <option value="inprogress">In Progress</option>
        <option value="completed">Completed</option>
      </select>
      <button type="submit">Add Task</button>
    </form>
  );
};

export default TaskForm;
