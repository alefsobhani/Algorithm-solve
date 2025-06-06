import React, { useEffect, useState } from 'react';
import axios from 'axios';
import Login from './components/Login';
import Register from './components/Register';
import Dashboard from './components/Dashboard';

const App = () => {
  const [user, setUser] = useState(null);
  const [view, setView] = useState('login');

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (token) {
      axios.get('/api/auth/me', { headers: { Authorization: `Bearer ${token}` } })
        .then(res => { setUser(res.data); setView('dashboard'); })
        .catch(() => setView('login'));
    }
  }, []);

  if (view === 'login') return <Login setView={setView} setUser={setUser} />;
  if (view === 'register') return <Register setView={setView} />;
  return <Dashboard user={user} setView={setView} />;
};

export default App;
