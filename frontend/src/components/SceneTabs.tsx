import React from 'react';
import './SceneTabs.css';

interface SceneTabsProps {
  activeScene: string;
  onSceneChange: (scene: string) => void;
  children: React.ReactNode;
}

export const SceneTabs: React.FC<SceneTabsProps> = ({
  activeScene,
  onSceneChange,
  children,
}) => {
  return (
    <div className="scene-container">
      <div className="scene-tabs">
        <button
          className={`scene-tab ${activeScene === 'pk-mode' ? 'active' : ''}`}
          onClick={() => onSceneChange('pk-mode')}
        >
          PK Mode
        </button>
        {/* Future scenes can be added here */}
      </div>
      <div className="scene-content">{children}</div>
    </div>
  );
};
