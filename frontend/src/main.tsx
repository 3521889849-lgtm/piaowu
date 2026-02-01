import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
// Ant Design v5 does not need manual CSS import for reset in most cases, 
// or it's built-in. If you need it, it's usually automatic.
// Removing the explicit import that causes error.

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
)
