import { useState, useEffect } from 'react';
import { LoginPage } from './components/LoginPage';
import { RegisterPage } from './components/RegisterPage';
import { PKModeScene } from './scenes/pk-mode/ui/PKModeScene';
import { OverlayApp } from './components/overlay/OverlayApp';
import { RefreshAuthToken, InitializeBBCoreClient, GetBBCoreURL } from '../wailsjs/go/main/App';
import type { User } from './shared/types';

import { Layout } from './components/layout/Layout';
import { Button } from './components/ui/button';
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert"
import { Terminal, Settings, Swords, Sticker, Gift } from 'lucide-react';
import { QuickConnect } from './components/dashboard/QuickConnect';
import { ModeCard } from './components/dashboard/ModeCard';
import { ConfigurationTab } from './components/configuration/ConfigurationTab';
import { MonitorTab } from './components/monitor/MonitorTab';
import { StickerDanceScene } from './scenes/sticker-dance/StickerDanceScene';
import { FreeModeScene } from './scenes/free-mode/FreeModeScene';

function App() {
  // Check if we are in overlay mode (via URL path or param)
  const isOverlay = window.location.pathname === '/overlay' || new URLSearchParams(window.location.search).has('overlay');

  if (isOverlay) {
    return <OverlayApp />;
  }

  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [authView, setAuthView] = useState<'login' | 'register'>('login');
  const [user, setUser] = useState<User | null>(null);
  const [accessToken, setAccessToken] = useState('');
  const [refreshToken, setRefreshToken] = useState('');
  const [activeTab, setActiveTab] = useState('pk-mode'); // Default to PK Mode for now
  const [sessionActive, setSessionActive] = useState(false);

  // Token refresh timer
  useEffect(() => {
    if (!isAuthenticated || !refreshToken) return;

    const interval = setInterval(async () => {
      try {
        const response = await RefreshAuthToken(refreshToken);
        setAccessToken(response.accessToken);
        setRefreshToken(response.refreshToken);
        console.log('Token refreshed successfully');
      } catch (error) {
        console.error('Token refresh failed:', error);
        handleLogout();
      }
    }, 50 * 60 * 1000);

    return () => clearInterval(interval);
  }, [isAuthenticated, refreshToken]);

  const handleLoginSuccess = async (
    newAccessToken: string,
    newRefreshToken: string,
    newUser: User
  ) => {
    // Persist tokens
    localStorage.setItem('auth_token', newAccessToken);
    localStorage.setItem('refresh_token', newRefreshToken);

    setAccessToken(newAccessToken);
    setRefreshToken(newRefreshToken);
    setUser(newUser);
    setIsAuthenticated(true);

    // Initialize BB-Core API client
    try {
      const bbCoreUrl = await GetBBCoreURL();
      await InitializeBBCoreClient(bbCoreUrl, newAccessToken);
      console.log('BB-Core API client initialized');
    } catch (error) {
      console.error('Failed to initialize BB-Core client:', error);
    }
  };

  const handleLogout = () => {
    // Clear tokens
    localStorage.removeItem('auth_token');
    localStorage.removeItem('refresh_token');

    setAccessToken('');
    setRefreshToken('');
    setUser(null);
    setIsAuthenticated(false);
    setActiveTab('pk-mode');
    setSessionActive(false);
  };

  const handleTabChange = async (newTab: string) => {
    if (sessionActive && activeTab === 'pk-mode' && newTab !== 'pk-mode') {
      const confirmed = window.confirm(
        'Active session detected. You must stop the current session before switching tabs.'
      );
      if (!confirmed) return;
      setSessionActive(false); // Ideally stop session via API too
    }
    setActiveTab(newTab);
  };

  if (!isAuthenticated) {
    // We can wrap Auth screens in a simple layout if desired, but for now keep full screen
    if (authView === 'register') {
      return (
        <RegisterPage
          onRegisterSuccess={handleLoginSuccess}
          onSwitchToLogin={() => setAuthView('login')}
        />
      );
    }
    return (
      <LoginPage
        onLoginSuccess={handleLoginSuccess}
        onSwitchToRegister={() => setAuthView('register')}
      />
    );
  }

  return (
    <Layout activeTab={activeTab} onTabChange={handleTabChange}>
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-4">
          <h1 className="text-2xl font-bold tracking-tight">
            {activeTab === 'dashboard' && 'Dashboard'}
            {activeTab === 'pk-mode' && 'PK Battle Arena'}
            {activeTab === 'monitor' && 'System Monitor'}
            {activeTab === 'configuration' && 'Configuration'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground mr-2">Welcome, {user?.username}</span>
          <Button variant="outline" size="sm" onClick={handleLogout}>Logout</Button>
        </div>
      </div>

      {activeTab === 'dashboard' && (
        <div className="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
            <div className="col-span-4">
              <QuickConnect onConnect={(roomId) => {
                setActiveTab('pk-mode');
                // TODO: Pass roomId to PK Mode to auto-start wizard with this ID
                console.log('Quick connect to:', roomId);
              }} />
            </div>
          </div>

          <div>
            <h3 className="text-xl font-semibold tracking-tight mb-4">Select Mode</h3>
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
              <ModeCard
                title="PK Battle"
                description="Team-based gift battles with real-time scoring and animations."
                icon={Swords}
                color="blue"
                onClick={() => setActiveTab('pk-mode')}
              />
              <ModeCard
                title="Sticker Dance"
                description="Interactive sticker triggering for dance challenges."
                icon={Sticker}
                color="pink"
                onClick={() => setActiveTab('sticker-dance')}
              />
              <ModeCard
                title="Free Mode"
                description="Casual streaming with basic gift tracking and alerts."
                icon={Gift}
                color="orange"
                onClick={() => setActiveTab('free-mode')}
              />
            </div>
          </div>

          <div className="rounded-xl border bg-card text-card-foreground shadow">
            <div className="p-6">
              <h3 className="font-semibold leading-none tracking-tight mb-4">Recent Rooms</h3>
              <div className="text-sm text-muted-foreground">No recent rooms found.</div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'pk-mode' && (
        <PKModeScene
          accessToken={accessToken}
          onSessionChange={setSessionActive}
        />
      )}

      {activeTab === 'monitor' && (
        <MonitorTab />
      )}

      {activeTab === 'sticker-dance' && (
        <StickerDanceScene />
      )}

      {activeTab === 'free-mode' && (
        <FreeModeScene />
      )}

      {activeTab === 'configuration' && (
        <ConfigurationTab />
      )}
    </Layout>
  );
}

export default App;
