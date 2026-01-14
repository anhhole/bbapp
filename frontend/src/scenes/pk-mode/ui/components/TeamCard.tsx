import React from 'react';
import { X, Plus } from 'lucide-react';
import type { Team, Idol } from '../../../../shared/types';
import './TeamCard.css';

interface TeamCardProps {
  team: Team;
  onUpdateTeam: (updatedTeam: Team) => void;
  onRemoveTeam: () => void;
  canRemove: boolean;
}

export const TeamCard: React.FC<TeamCardProps> = ({
  team,
  onUpdateTeam,
  onRemoveTeam,
  canRemove,
}) => {
  const handleTeamNameChange = (name: string) => {
    onUpdateTeam({ ...team, name });
  };

  const handleTeamGiftChange = (bindingGift: string) => {
    onUpdateTeam({ ...team, bindingGift });
  };

  const handleAddIdol = () => {
    const newIdolId = `idol-${Date.now()}`;
    const newIdol: Idol = {
      id: newIdolId,
      bigoId: '',
      bigoRoomId: '',
      name: `Idol ${team.streamers.length + 1}`,
      bindingGift: team.bindingGift,
    };
    onUpdateTeam({
      ...team,
      streamers: [...team.streamers, newIdol],
    });
  };

  const handleUpdateIdol = (index: number, updatedIdol: Idol) => {
    const updatedIdols = [...team.streamers];
    updatedIdols[index] = updatedIdol;
    onUpdateTeam({ ...team, streamers: updatedIdols });
  };

  const handleRemoveIdol = (index: number) => {
    if (team.streamers.length <= 1) {
      alert('Each team must have at least one idol');
      return;
    }
    const updatedIdols = team.streamers.filter((_, i) => i !== index);
    onUpdateTeam({ ...team, streamers: updatedIdols });
  };

  return (
    <div className="team-card">
      <div className="team-card-header">
        <h3>{team.name}</h3>
        {canRemove && (
          <button
            className="remove-team-btn"
            onClick={onRemoveTeam}
            title="Remove team"
          >
            <X size={18} />
          </button>
        )}
      </div>

      <div className="team-card-body">
        <div className="form-row">
          <label>Team Name:</label>
          <input
            type="text"
            value={team.name}
            onChange={(e) => handleTeamNameChange(e.target.value)}
            placeholder="Enter team name"
          />
        </div>

        <div className="form-row">
          <label>Binding Gift:</label>
          <input
            type="text"
            value={team.bindingGift}
            onChange={(e) => handleTeamGiftChange(e.target.value)}
            placeholder="e.g., Rose, Diamond"
          />
        </div>

        <div className="streamers-section">
          <div className="streamers-header">
            <label>Idols:</label>
            <button className="add-streamer-btn" onClick={handleAddIdol}>
              <Plus size={16} /> Add Idol
            </button>
          </div>

          <div className="streamers-list">
            {team.streamers.map((idol, index) => (
              <div key={idol.id} className="streamer-card">
                <div className="streamer-header">
                  <span className="streamer-title">{idol.name}</span>
                  <button
                    className="remove-streamer-btn"
                    onClick={() => handleRemoveIdol(index)}
                    disabled={team.streamers.length <= 1}
                    title="Remove idol"
                  >
                    <X size={14} />
                  </button>
                </div>

                <div className="streamer-fields">
                  <div className="form-row">
                    <label>Name:</label>
                    <input
                      type="text"
                      value={idol.name}
                      onChange={(e) =>
                        handleUpdateIdol(index, {
                          ...idol,
                          name: e.target.value,
                        })
                      }
                      placeholder="Idol name"
                    />
                  </div>

                  <div className="form-row">
                    <label>Bigo Room ID:</label>
                    <input
                      type="text"
                      value={idol.bigoRoomId}
                      onChange={(e) =>
                        handleUpdateIdol(index, {
                          ...idol,
                          bigoRoomId: e.target.value,
                        })
                      }
                      placeholder="e.g., room123"
                    />
                  </div>

                  <div className="form-row">
                    <label>Binding Gift:</label>
                    <input
                      type="text"
                      value={idol.bindingGift}
                      onChange={(e) =>
                        handleUpdateIdol(index, {
                          ...idol,
                          bindingGift: e.target.value,
                        })
                      }
                      placeholder="e.g., Rose"
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
};
