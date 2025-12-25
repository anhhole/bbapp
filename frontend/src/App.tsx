import { useState, useEffect } from 'react';
import { LoginPage } from './components/LoginPage';
import { SceneTabs } from './components/SceneTabs';
import { PKModeScene } from './scenes/pk-mode/ui/PKModeScene';
import { RefreshToken } from '../wailsjs/go/main/App';
import type { User } from './shared/types';
import './App.css';

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState('');
  const [refreshToken, setRefreshToken] = useState('');
  const [activeScene, setActiveScene] = useState('pk-mode');
  const [sessionActive, setSessionActive] = useState(false);

  // Token refresh timer
  useEffect(() => {
    if (!isAuthenticated || !refreshToken) return;

    // Refresh token every 50 minutes (assuming 60 min expiry)
    const interval = setInterval(async () => {
      try {
        const response = await RefreshToken(refreshToken);
        setAccessToken(response.accessToken);
        setRefreshToken(response.refreshToken);
        console.log('Token refreshed successfully');
      } catch (error) {
        console.error('Token refresh failed:', error);
        handleLogout();
      }
    }, 50 * 60 * 1000); // 50 minutes

    return () => clearInterval(interval);
  }, [isAuthenticated, refreshToken]);

  const handleLoginSuccess = (
    newAccessToken: string,
    newRefreshToken: string,
    newUser: User
  ) => {
    setAccessToken(newAccessToken);
    setRefreshToken(newRefreshToken);
    setUser(newUser);
    setIsAuthenticated(true);
  };

  const handleLogout = () => {
    setAccessToken('');
    setRefreshToken('');
    setUser(null);
    setIsAuthenticated(false);
    setActiveScene('pk-mode');
    setSessionActive(false);
  };

  const handleSceneChange = async (newScene: string) => {
    if (sessionActive) {
      const confirmed = window.confirm(
        'Active session detected. You must stop the current session before switching scenes.'
      );
      if (!confirmed) return;
      // TODO: Stop session here
      // await StopPKSession('USER_SWITCHED_SCENES');
      setSessionActive(false);
    }
    setActiveScene(newScene);
  };

  if (!isAuthenticated) {
    return <LoginPage onLoginSuccess={handleLoginSuccess} />;
  }

  return (
    <div className="app">
      <header className="app-header">
        <h1>BBapp - PK Session Manager</h1>
        <div className="user-info">
          <span>Welcome, {user?.username}</span>
          <button onClick={handleLogout} className="logout-btn">
            Logout
          </button>
        </div>
      </header>

      <SceneTabs activeScene={activeScene} onSceneChange={handleSceneChange}>
        {activeScene === 'pk-mode' && (
          <PKModeScene
            accessToken={accessToken}
            onSessionChange={setSessionActive}
          />
        )}
      </SceneTabs>
    </div>
  );
}

export default App;
