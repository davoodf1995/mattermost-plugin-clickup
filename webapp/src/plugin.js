import {FormattedMessage} from 'react-intl';
import {Client4} from 'mattermost-redux/client';
import {getPluginURL} from './utils';
import TasksRHS, {createTasksRHS} from './components/tasks_rhs';

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

        const TasksRHSConnected = createTasksRHS(store);

        const rhs = registry.registerRightHandSidebarComponent({
            title: (
                <FormattedMessage
                    id='plugin.rhs.title'
                    defaultMessage='ClickUp Tasks'
                />
            ),
            component: TasksRHSConnected,
        });

        const openTasks = (channelId) => {
            const activeChannelId = channelId || store.getState().entities.channels.currentChannelId;
            if (activeChannelId) {
                store.dispatch(rhs.toggleRHSPlugin);
            }
        };

        registry.registerAppBarComponent({
            iconUrl: `${pluginURL}/public/clickup.png`,
            action: openTasks,
            tooltipText: (
                <FormattedMessage
                    id='plugin.app_bar'
                    defaultMessage='ClickUp Tasks'
                />
            ),
        });

        registry.registerChannelHeaderButtonAction(
            clickupIcon,
            openTasks,
            'ClickUp Tasks',
        );
    }
}
