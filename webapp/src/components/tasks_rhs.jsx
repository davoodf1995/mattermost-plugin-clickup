import {getPluginURL} from '../utils';

const React = window.React;
const PropTypes = window.PropTypes;

export default class TasksRHS extends React.PureComponent {
    static propTypes = {
        channelId: PropTypes.string,
    }

    constructor(props) {
        super(props);
        this.state = {
            loading: true,
            error: '',
            tasks: [],
            link: null,
        };
    }

    componentDidMount() {
        this.loadTasks();
    }

    componentDidUpdate(prevProps) {
        if (prevProps.channelId !== this.props.channelId) {
            this.loadTasks();
        }
    }

    loadTasks = async () => {
        const channelId = this.props.channelId;
        if (!channelId) {
            this.setState({loading: false, error: 'No channel selected', tasks: []});
            return;
        }

        this.setState({loading: true, error: ''});

        try {
            const response = await fetch(`${getPluginURL()}/api/tasks?channel_id=${channelId}`, {
                credentials: 'same-origin',
            });

            if (!response.ok) {
                const text = await response.text();
                throw new Error(text || 'Failed to load tasks');
            }

            const data = await response.json();
            this.setState({
                loading: false,
                tasks: data.tasks || [],
                link: data.link || null,
            });
        } catch (error) {
            this.setState({
                loading: false,
                error: error.message,
                tasks: [],
            });
        }
    }

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
                {!loading && error && (
                    <div>
                        <p>{error}</p>
                        <p style={{fontSize: '12px'}}>
                            Link a list with <code>/clickup link &lt;list_id&gt;</code> or set a default list in plugin settings.
                        </p>
                    </div>
                )}
                {!loading && !error && tasks.length === 0 && (
                    <div>No open tasks.</div>
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
}
