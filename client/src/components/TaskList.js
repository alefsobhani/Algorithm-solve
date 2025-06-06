import React from 'react';
import axios from 'axios';

const TaskList = ({ tasks, refresh }) => {
  const token = localStorage.getItem('token');

  const handleDelete = async id => {
    await axios.delete(`/api/tasks/${id}`, { headers: { Authorization: `Bearer ${token}` } });
    refresh();
  };

  return (
    <ul>
      {tasks.map(task => (
        <li key={task._id}>
          {task.title} - {task.status}
          <button onClick={() => handleDelete(task._id)}>Delete</button>
        </li>
      ))}
    </ul>
  );
};

export default TaskList;
