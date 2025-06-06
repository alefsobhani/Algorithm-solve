import React, { useEffect, useState } from 'react';
import axios from 'axios';

const Comments = ({ taskId }) => {
  const [comments, setComments] = useState([]);
  const [text, setText] = useState('');
  const token = localStorage.getItem('token');

  const fetchComments = async () => {
    const { data } = await axios.get(`/api/tasks/${taskId}/comments`, { headers: { Authorization: `Bearer ${token}` } });
    setComments(data);
  };

  useEffect(() => { if (taskId) fetchComments(); }, [taskId]);

  const handleSubmit = async e => {
    e.preventDefault();
    await axios.post(`/api/tasks/${taskId}/comments`, { text }, { headers: { Authorization: `Bearer ${token}` } });
    setText('');
    fetchComments();
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
