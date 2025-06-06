import React, { useState } from 'react';
import axios from 'axios';

const Register = ({ setView }) => {
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');

  const handleSubmit = async e => {
    e.preventDefault();
    try {
      await axios.post('/api/auth/register', { name, email, password });
      alert('Registered! Please login.');
      setView('login');
    } catch(err) {
      alert(err.response.data.message);
    }
  };

  return (
    <div className="container">
      <h2>Register</h2>
      <form onSubmit={handleSubmit}>
        <input placeholder="Name" value={name} onChange={e => setName(e.target.value)} />
        <input placeholder="Email" value={email} onChange={e => setEmail(e.target.value)} />
        <input type="password" placeholder="Password" value={password} onChange={e => setPassword(e.target.value)} />
        <button type="submit">Register</button>
      </form>
      <p>Already have an account? <button onClick={() => setView('login')}>Login</button></p>
    </div>
  );
};

export default Register;
