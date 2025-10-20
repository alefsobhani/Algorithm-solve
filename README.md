# Task Management System

This repository contains a simple task management web application built with **Node.js/Express** for the backend and **React** for the frontend.

## Features
- User authentication with JWT
- Password reset via email
- Task CRUD operations
- Role based access (admin, user)
- Comment on tasks and tag users
- Responsive frontend with React

## Getting Started

### Backend
1. Navigate to `server` and install dependencies:
   ```bash
   npm install
   ```
2. Copy `.env.example` to `.env` and update the values.
3. Run the server:
   ```bash
   node server.js
   ```

### Frontend
1. Navigate to `client` and install dependencies:
   ```bash
   npm install
   ```
2. Start the React app:
   ```bash
   npm start
   ```

## Deployment

### Heroku (Backend)
1. Create a Heroku app and add a MongoDB add-on or connection string.
2. Set the environment variables from `.env` in Heroku settings.
3. Push the contents of the `server` folder to Heroku or configure a multi-buildpack.

### Netlify (Frontend)
1. Create a new site from the `client` directory.
2. Set the build command to `npm run build` and the publish directory to `build`.
3. Ensure the API endpoint URLs in the frontend point to your Heroku backend.

### Rehab Manager (Visual Basic)
A simple Visual Basic console application is available in `Rehab_Manager`.
Compile with a VB compiler such as `vbc` and run the resulting executable.

## License
MIT
