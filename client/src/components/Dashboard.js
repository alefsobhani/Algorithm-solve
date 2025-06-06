import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { useThemeToggle } from '../contexts/ThemeContext';
import api from '../services/api';
import TaskForm from './TaskForm';
import TaskList from './TaskList';
import Comments from './Comments';
import { Pie } from 'react-chartjs-2';
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from 'chart.js';
ChartJS.register(ArcElement, Tooltip, Legend);

const Dashboard = () => {
  const { logout } = useAuth();
  const toggleTheme = useThemeToggle();
  const navigate = useNavigate();
  const [tasks, setTasks] = useState([]);
  const [priority, setPriority] = useState('');
  const [status, setStatus] = useState('');
  const [sort, setSort] = useState('');
  const [selectedTask, setSelectedTask] = useState(null);
  const completed = tasks.filter(t => t.status === 'completed').length;
  const progress = tasks.length ? Math.round((completed / tasks.length) * 100) : 0;
  const chartData = {
    labels: ['Completed', 'Remaining'],
    datasets: [{ data: [completed, tasks.length - completed], backgroundColor: ['#4caf50', '#f44336'] }]
  };

  const fetchTasks = async () => {
    const params = new URLSearchParams();
    if (priority) params.append('priority', priority);
    if (status) params.append('status', status);
    if (sort) params.append('sort', sort);
    const { data } = await api.get(`/tasks?${params.toString()}`);
    setTasks(data);
  };

  useEffect(() => { fetchTasks(); }, [priority, status, sort]);

  return (
    <div className="container">
      <h2>Dashboard</h2>
      <button onClick={toggleTheme}>Toggle Theme</button>
      <button onClick={() => { logout(); navigate('/login'); }}>Logout</button>
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
      <Pie data={chartData} />
      <TaskForm refresh={fetchTasks} />
      <TaskList tasks={tasks} refresh={fetchTasks} onSelect={setSelectedTask} />
      <Comments taskId={selectedTask} />
    </div>
  );
};

export default Dashboard;
