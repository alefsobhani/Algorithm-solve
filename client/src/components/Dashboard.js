import React, { useEffect, useState } from 'react';
import axios from 'axios';
import TaskForm from './TaskForm';
import TaskList from './TaskList';
import Comments from './Comments';

const Dashboard = ({ user, setView }) => {
  const [tasks, setTasks] = useState([]);
  const [priority, setPriority] = useState('');
  const [status, setStatus] = useState('');
  const [sort, setSort] = useState('');
  const [selectedTask, setSelectedTask] = useState(null);
  const completed = tasks.filter(t => t.status === 'completed').length;
  const progress = tasks.length ? Math.round((completed / tasks.length) * 100) : 0;

  const fetchTasks = async () => {
    const token = localStorage.getItem('token');
    const params = new URLSearchParams();
    if (priority) params.append('priority', priority);
    if (status) params.append('status', status);
    if (sort) params.append('sort', sort);
    const { data } = await axios.get(`/api/tasks?${params.toString()}`, { headers: { Authorization: `Bearer ${token}` } });
    setTasks(data);
  };

  useEffect(() => { fetchTasks(); }, [priority, status, sort]);

  return (
    <div className="container">
      <h2>Dashboard</h2>
      <button onClick={() => { localStorage.removeItem('token'); setView('login'); }}>Logout</button>
      <div className="filters">
        <select value={priority} onChange={e => setPriority(e.target.value)}>
          <option value="">All Priorities</option>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
        </select>
        <select value={status} onChange={e => setStatus(e.target.value)}>
          <option value="">All Statuses</option>
          <option value="todo">To Do</option>
          <option value="inprogress">In Progress</option>
          <option value="completed">Completed</option>
        </select>
        <select value={sort} onChange={e => setSort(e.target.value)}>
          <option value="">No Sort</option>
          <option value="dueDate">Due Date</option>
          <option value="priority">Priority</option>
        </select>
      </div>
      <p>Progress: {progress}% completed</p>
      <TaskForm refresh={fetchTasks} />
      <TaskList tasks={tasks} refresh={fetchTasks} onSelect={setSelectedTask} />
      <Comments taskId={selectedTask} />
    </div>
  );
};

export default Dashboard;
