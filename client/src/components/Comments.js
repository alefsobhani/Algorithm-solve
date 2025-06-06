import React, { useEffect, useState } from 'react';
import api from '../services/api';
import { io } from 'socket.io-client';

const socket = io();

const Comments = ({ taskId }) => {
  const [comments, setComments] = useState([]);
  const [text, setText] = useState('');
  const fetchComments = async () => {
    const { data } = await api.get(`/tasks/${taskId}/comments`);
    setComments(data);
  };

  useEffect(() => { if (taskId) fetchComments(); }, [taskId]);
  useEffect(() => {
    if (!taskId) return;
    const handler = c => { if (c.task === taskId) setComments(prev => [...prev, c]); };
    socket.on('comment', handler);
    return () => { socket.off('comment', handler); };
  }, [taskId]);

  const handleSubmit = async e => {
    e.preventDefault();
    await api.post(`/tasks/${taskId}/comments`, { text });
    setText('');
  };

  if (!taskId) return null;

  return (
    <div>
      <h3>Comments</h3>
      <ul>
        {comments.map(c => (
          <li key={c._id}>{c.author?.name}: {c.text}</li>
        ))}
      </ul>
      <form onSubmit={handleSubmit}>
        <input value={text} onChange={e => setText(e.target.value)} placeholder="Add comment" />
        <button type="submit">Add</button>
      </form>
    </div>
  );
};

export default Comments;
