import React from 'react';
import { X, Plus } from 'lucide-react';
import type { Team, Streamer } from '../../../../shared/types';
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

  const handleAddStreamer = () => {
    const newStreamerId = Date.now();
    const newStreamer: Streamer = {
      streamerId: newStreamerId,
      bigoId: '',
      bigoRoomId: '',
      name: `Streamer ${team.streamers.length + 1}`,
      bindingGift: team.bindingGift,
    };
    onUpdateTeam({
      ...team,
      streamers: [...team.streamers, newStreamer],
    });
  };

  const handleUpdateStreamer = (index: number, updatedStreamer: Streamer) => {
    const updatedStreamers = [...team.streamers];
    updatedStreamers[index] = updatedStreamer;
    onUpdateTeam({ ...team, streamers: updatedStreamers });
  };

  const handleRemoveStreamer = (index: number) => {
    if (team.streamers.length <= 1) {
      alert('Each team must have at least one streamer');
      return;
    }
    const updatedStreamers = team.streamers.filter((_, i) => i !== index);
    onUpdateTeam({ ...team, streamers: updatedStreamers });
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
            <label>Streamers:</label>
            <button className="add-streamer-btn" onClick={handleAddStreamer}>
              <Plus size={16} /> Add Streamer
            </button>
          </div>

          <div className="streamers-list">
            {team.streamers.map((streamer, index) => (
              <div key={streamer.streamerId} className="streamer-card">
                <div className="streamer-header">
                  <span className="streamer-title">{streamer.name}</span>
                  <button
                    className="remove-streamer-btn"
                    onClick={() => handleRemoveStreamer(index)}
                    disabled={team.streamers.length <= 1}
                    title="Remove streamer"
                  >
                    <X size={14} />
                  </button>
                </div>

                <div className="streamer-fields">
                  <div className="form-row">
                    <label>Name:</label>
                    <input
                      type="text"
                      value={streamer.name}
                      onChange={(e) =>
                        handleUpdateStreamer(index, {
                          ...streamer,
                          name: e.target.value,
                        })
                      }
                      placeholder="Streamer name"
                    />
                  </div>

                  <div className="form-row">
                    <label>Bigo Room ID:</label>
                    <input
                      type="text"
                      value={streamer.bigoRoomId}
                      onChange={(e) =>
                        handleUpdateStreamer(index, {
                          ...streamer,
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
                      value={streamer.bindingGift}
                      onChange={(e) =>
                        handleUpdateStreamer(index, {
                          ...streamer,
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
