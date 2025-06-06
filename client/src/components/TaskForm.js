import React, { useState } from 'react';
import axios from 'axios';

const TaskForm = ({ refresh }) => {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');

  const handleSubmit = async e => {
    e.preventDefault();
    const token = localStorage.getItem('token');
    await axios.post('/api/tasks', { title, description }, { headers: { Authorization: `Bearer ${token}` } });
    setTitle('');
    setDescription('');
    refresh();
  };

  return (
    <form onSubmit={handleSubmit}>
      <input placeholder="Title" value={title} onChange={e => setTitle(e.target.value)} />
      <input placeholder="Description" value={description} onChange={e => setDescription(e.target.value)} />
      <button type="submit">Add Task</button>
    </form>
  );
};

export default TaskForm;
