import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/channels';

import {getPluginURL} from '../utils';

const React = window.React;

function normalizeChannelId(value) {
    if (typeof value === 'string' && value.trim() !== '') {
        return value.trim();
    }
    if (value && typeof value === 'object' && typeof value.id === 'string' && value.id !== '') {
        return value.id;
    }
    return '';
}

export function createTasksRHS(store, getPinnedChannelId) {
    const resolveChannelId = () => {
        const fromStore = getCurrentChannelId(store.getState()) || '';
        const pinned = getPinnedChannelId ? getPinnedChannelId() : '';
        return fromStore || pinned || '';
    };

    return class TasksRHS extends React.PureComponent {
        constructor(props) {
            super(props);
            this.state = {
                loading: true,
                error: '',
                tasks: [],
                link: null,
                channelId: '',
            };
            this.lastLoadedChannelId = '';
        }

        componentDidMount() {
            this.unsubscribe = store.subscribe(this.handleStoreChange);
            this.loadTasks();
        }

        componentWillUnmount() {
            if (this.unsubscribe) {
                this.unsubscribe();
            }
        }

        handleStoreChange = () => {
            const channelId = resolveChannelId();
            if (channelId && channelId !== this.lastLoadedChannelId) {
                this.loadTasks();
            }
        };

        loadTasks = async () => {
            const channelId = resolveChannelId();
            if (!channelId) {
                this.setState({
                    loading: false,
                    error: 'no_channel',
                    tasks: [],
                    link: null,
                    channelId: '',
                });
                return;
            }

            this.setState({loading: true, error: '', channelId});

            try {
                const response = await fetch(`${getPluginURL()}/api/tasks?channel_id=${encodeURIComponent(channelId)}`, {
                    credentials: 'same-origin',
                });

                if (!response.ok) {
                    const text = await response.text();
                    throw new Error(text || 'Failed to load tasks');
                }

                const data = await response.json();
                this.lastLoadedChannelId = channelId;
                this.setState({
                    loading: false,
                    tasks: data.tasks || [],
                    link: data.link || null,
                    channelId,
                });
            } catch (error) {
                this.setState({
                    loading: false,
                    error: error.message,
                    tasks: [],
                    channelId,
                });
            }
        };

        renderTask(task) {
            const assignees = (task.assignees || []).map((a) => a.username || a.email).join(', ') || 'unassigned';
            const status = task.status?.status || 'open';

            return (
                <div
                    key={task.id}
                    style={{
                        border: '1px solid rgba(var(--center-channel-color-rgb), 0.16)',
                        borderRadius: '4px',
                        padding: '10px',
                        marginBottom: '8px',
                    }}
                >
                    <a
                        href={task.url}
                        target='_blank'
                        rel='noopener noreferrer'
                        style={{fontWeight: 600}}
                    >
                        {task.name}
                    </a>
                    <div style={{fontSize: '12px', opacity: 0.72, marginTop: '4px'}}>
                        {status} · {assignees}
                    </div>
                </div>
            );
        }

        render() {
            const {loading, error, tasks, link} = this.state;

            return (
                <div style={{padding: '12px'}}>
                    <div style={{marginBottom: '12px'}}>
                        <strong>ClickUp Tasks</strong>
                        {link?.list_id && (
                            <div style={{fontSize: '12px', opacity: 0.72}}>
                                List: {link.list_name || link.list_id}
                            </div>
                        )}
                    </div>

                    {loading && <div>Loading...</div>}
                    {!loading && error === 'no_channel' && (
                        <div>
                            <p>Open a <strong>channel</strong> first, then open ClickUp from the <strong>channel header</strong> (top of the chat).</p>
                            <p style={{fontSize: '12px'}}>
                                The app-bar icon needs an active channel. Link a list with <code>/clickup link &lt;url&gt;</code> inside that channel.
                            </p>
                        </div>
                    )}
                    {!loading && error && error !== 'no_channel' && (
                        <div>
                            <p>{error}</p>
                            <p style={{fontSize: '12px'}}>
                                Run <code>/clickup link &lt;url&gt;</code> in this channel, or <code>/clickup lists</code> to find list IDs.
                            </p>
                        </div>
                    )}
                    {!loading && !error && tasks.length > 0 && (
                        <div style={{fontSize: '12px', opacity: 0.72, marginBottom: '8px'}}>
                            {tasks.length} open task{tasks.length === 1 ? '' : 's'}
                        </div>
                    )}
                    {!loading && !error && tasks.length === 0 && (
                        <div>No open tasks in the linked list/view.</div>
                    )}
                    {!loading && !error && tasks.map((task) => this.renderTask(task))}

                    <button
                        className='btn btn-link'
                        onClick={this.loadTasks}
                        style={{marginTop: '8px'}}
                    >
                        Refresh
                    </button>
                </div>
            );
        }
    };
}

export {normalizeChannelId};
