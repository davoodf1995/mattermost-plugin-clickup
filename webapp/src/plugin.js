import {FormattedMessage} from 'react-intl';
import {getCurrentChannelId} from 'mattermost-redux/selectors/entities/channels';
import {Client4} from 'mattermost-redux/client';
import {getPluginURL} from './utils';
import {createTasksRHS, normalizeChannelId} from './components/tasks_rhs';

const React = window.React;

const clickupIcon = (
    <img
        src={`${getPluginURL()}/public/clickup.png`}
        alt='ClickUp'
        style={{width: '24px', height: '24px'}}
    />
);

export default class ClickUpPlugin {
    initialize(registry, store) {
        const pluginURL = getPluginURL();
        let pinnedChannelId = '';

        const pinChannel = (channelLike) => {
            const id = normalizeChannelId(channelLike) || getCurrentChannelId(store.getState()) || '';
            if (id) {
                pinnedChannelId = id;
            }
            return id;
        };

        registry.registerPostDropdownMenuAction(
            <FormattedMessage
                id='plugin.post_action'
                defaultMessage='Create ClickUp Task'
            />,
            (postId) => {
                const post = store.getState().entities.posts.posts[postId];
                if (!post) {
                    return;
                }

                const title = (post.message || '').split('\n')[0].trim().slice(0, 100);
                if (!title) {
                    return;
                }

                Client4.executeCommand(post.channel_id, `/clickup task ${title}`);
            },
        );

        const TasksRHS = createTasksRHS(store, () => pinnedChannelId);

        const rhs = registry.registerRightHandSidebarComponent({
            title: (
                <FormattedMessage
                    id='plugin.rhs.title'
                    defaultMessage='ClickUp Tasks'
                />
            ),
            component: TasksRHS,
        });

        const openTasks = (channelLike) => {
            const channelId = pinChannel(channelLike);
            if (!channelId) {
                return;
            }
            store.dispatch(rhs.showRHSPlugin);
        };

        registry.registerAppBarComponent({
            iconUrl: `${pluginURL}/public/clickup.png`,
            action: (channelLike) => openTasks(channelLike),
            tooltipText: (
                <FormattedMessage
                    id='plugin.app_bar'
                    defaultMessage='ClickUp Tasks'
                />
            ),
        });

        registry.registerChannelHeaderButtonAction(
            clickupIcon,
            (channel) => openTasks(channel),
            'ClickUp Tasks',
            'ClickUp Tasks',
        );
    }
}
