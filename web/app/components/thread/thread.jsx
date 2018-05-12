import { h, Component } from 'preact';

import Comment from 'components/comment';

export default class Thread extends Component {
  constructor(props) {
    super(props);

    this.state = {
      collapsed: false,
    };

    this.onCollapseToggle = this.onCollapseToggle.bind(this);
  }


  onCollapseToggle() {
    this.setState({ collapsed: !this.state.collapsed });
  }

  render(props, { collapsed }) {
    const { data: { comment, replies = [] }, mix, mods = {}, onReplyClick } = props;

    return (
      <div className={b('thread', props)}>
        <Comment
          data={comment}
          mods={{ level: mods.level, collapsed, collapsible: !!replies.length }}
          onReply={props.onReply}
          onReplyClick={onReplyClick}
          onCollapseToggle={this.onCollapseToggle}
        />

        {
          !collapsed && !!replies.length && replies.map(thread => (
            <Thread
              data={thread}
              mods={{ level: mods.level < 5 ? mods.level + 1 : mods.level }}
              onReply={props.onReply}
              onReplyClick={onReplyClick}
            />
          ))
        }
      </div>
    );
  }
}
