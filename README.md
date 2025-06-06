# Task Management System

This repository contains a basic project management web application built with **Node.js/Express** for the backend and **React** for the frontend.

## Features
- User authentication with JWT
- Password reset via email
- Task CRUD operations with due dates, priorities and statuses
- Role based access (admin, user)
- Comment on tasks and tag users
- Filter and sort tasks by priority, status and due date
- Inline status updates and comment threads
- Progress indicator for completed tasks
- Responsive frontend with React Router and Context API
- Basic project boards
- Helmet and rate limiting for security
- Dark/light theme toggle
- Real-time comment updates via Socket.IO

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
4. Run tests:
   ```bash
   npm test
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
3. Run tests:
   ```bash
   npm test
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

## License
MIT
