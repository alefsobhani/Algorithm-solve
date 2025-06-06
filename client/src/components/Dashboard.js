import React, { useEffect, useState } from 'react';
import axios from 'axios';
import TaskForm from './TaskForm';
import TaskList from './TaskList';

const Dashboard = ({ user, setView }) => {
  const [tasks, setTasks] = useState([]);

  const fetchTasks = async () => {
    const token = localStorage.getItem('token');
    const { data } = await axios.get('/api/tasks', { headers: { Authorization: `Bearer ${token}` } });
    setTasks(data);
  };

  useEffect(() => { fetchTasks(); }, []);

  return (
    <div className="container">
      <h2>Dashboard</h2>
      <button onClick={() => { localStorage.removeItem('token'); setView('login'); }}>Logout</button>
      <TaskForm refresh={fetchTasks} />
      <TaskList tasks={tasks} refresh={fetchTasks} />
    </div>
  );
};

export default Dashboard;
