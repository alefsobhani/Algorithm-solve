import React, { useEffect, useState } from 'react';
import api from '../services/api';

const Projects = () => {
  const [projects, setProjects] = useState([]);

  const fetchProjects = async () => {
    const { data } = await api.get('/projects');
    setProjects(data);
  };

  useEffect(() => { fetchProjects(); }, []);

  return (
    <div className="container">
      <h2>Projects</h2>
      <ul>
        {projects.map(p => (
          <li key={p._id}>{p.title}</li>
        ))}
      </ul>
    </div>
  );
};

export default Projects;
