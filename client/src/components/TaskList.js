import React from 'react';
import axios from 'axios';

const TaskList = ({ tasks, refresh, onSelect }) => {
  const token = localStorage.getItem('token');

  const handleDelete = async id => {
    await axios.delete(`/api/tasks/${id}`, { headers: { Authorization: `Bearer ${token}` } });
    refresh();
  };

  const handleStatusChange = async (id, status) => {
    await axios.put(`/api/tasks/${id}`, { status }, { headers: { Authorization: `Bearer ${token}` } });
    refresh();
  };

  return (
    <table>
      <thead>
        <tr>
          <th>Title</th>
          <th>Priority</th>
          <th>Status</th>
          <th>Due</th>
          <th>Actions</th>
          <th>Comments</th>
        </tr>
      </thead>
      <tbody>
        {tasks.map(task => (
          <tr key={task._id}>
            <td>{task.title}</td>
            <td>{task.priority}</td>
            <td>
              <select value={task.status} onChange={e => handleStatusChange(task._id, e.target.value)}>
                <option value="todo">To Do</option>
                <option value="inprogress">In Progress</option>
                <option value="completed">Completed</option>
              </select>
            </td>
            <td>{task.dueDate ? new Date(task.dueDate).toLocaleDateString() : ''}</td>
            <td><button onClick={() => handleDelete(task._id)}>Delete</button></td>
            <td><button onClick={() => onSelect(task._id)}>Comments</button></td>
          </tr>
        ))}
      </tbody>
    </table>
  );
};

export default TaskList;
