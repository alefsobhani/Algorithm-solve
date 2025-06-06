import React, { createContext, useContext, useMemo, useState } from 'react';
import { ThemeProvider, createTheme } from '@mui/material/styles';

const ThemeToggleContext = createContext(() => {});

export const CustomThemeProvider = ({ children }) => {
  const [mode, setMode] = useState('light');
  const theme = useMemo(() => createTheme({ palette: { mode } }), [mode]);

  const toggle = () => setMode(m => (m === 'light' ? 'dark' : 'light'));

  return (
    <ThemeToggleContext.Provider value={toggle}>
      <ThemeProvider theme={theme}>{children}</ThemeProvider>
    </ThemeToggleContext.Provider>
  );
};

export const useThemeToggle = () => useContext(ThemeToggleContext);
