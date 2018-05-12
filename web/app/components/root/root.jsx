import { h, Component } from 'preact';
import api from 'common/api';

import { BASE_URL, NODE_ID, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants';
import { url } from 'common/settings';
import store from 'common/store';

import AuthPanel from 'components/auth-panel';
import BlockedUsers from 'components/blocked-users';
import Comment from 'components/comment';
import Input from 'components/input';
import Preloader from 'components/preloader';
import Thread from 'components/thread';

const LS_SORT_KEY = '__remarkSort';

export default class Root extends Component {
  constructor(props) {
    super(props);

    let sort;

    try {
      sort = localStorage.getItem(LS_SORT_KEY);
    } catch(e) {
      sort = '-score';
    }

    this.state = {
      isLoaded: false,
      isCommentsListLoading: false,
      user: {},
      replyingCommentId: null,
      sort,
    };

    this.addComment = this.addComment.bind(this);
    this.onSignIn = this.onSignIn.bind(this);
    this.onSignOut = this.onSignOut.bind(this);
    this.onBlockedUsersShow = this.onBlockedUsersShow.bind(this);
    this.onBlockedUsersHide = this.onBlockedUsersHide.bind(this);
    this.onReplyClick = this.onReplyClick.bind(this);
    this.onSortChange = this.onSortChange.bind(this);
    this.onUnblockSomeone = this.onUnblockSomeone.bind(this);
    this.checkUrlHash = this.checkUrlHash.bind(this);
  }

  componentWillMount() {
    store.onUpdate('comments', comments => this.setState({ comments }));
    store.onUpdate('replyingCommentId', id => this.setState({ replyingCommentId: id }));
  }

  componentDidMount() {
    const { sort } = this.state;

    api.getUser()
      .then(data => store.set('user', data))
      .catch(() => store.set('user', {}))
      .finally(() => {
        api.find({ sort, url })
          .then(({ comments = [] } = {}) => store.set('comments', comments))
          .catch(() => store.set('comments', []))
          .finally(() => {
            this.setState({
              isLoaded: true,
              user: store.get('user'),
            });

            setTimeout(this.checkUrlHash);
          });

        api.getConfig().then(config => {
          store.set('config', config);
          this.setState({ config });
        })
      });
  }

  checkUrlHash() {
    if (window.location.hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      const comment = document.querySelector(window.location.hash);

      if (comment) {
        setTimeout(() => {
          comment.scrollIntoView();
          window.scrollTo(window.scrollX, 0); // sometimes smth goes wrong and iframe scrolls; we return it here
        }, 500);
      }
    }
  }

  onSignOut() {
    api.logOut().then(() => {
      store.set('user', {});
      this.setState({ user: {} });
    });
  }

  onSignIn(provider) {
    const newWindow = window.open(`${BASE_URL}/auth/${provider}/login?from=${encodeURIComponent(location.href)}`);

    let secondsPass = 0;
    const checkMsDelay = 200;
    const checkInterval = setInterval(() => {
      secondsPass += checkMsDelay;

      if (newWindow.location.origin === location.origin || secondsPass > 30000) {
        clearInterval(checkInterval);
        secondsPass = 0;
        newWindow.close();

        api.getUser()
          .then(user => {
            store.set('user', user);
            this.setState({ user });
          })
          .catch(() => {}) // TODO: we need to handle it and write error to user
      }
    }, checkMsDelay);
  }

  onBlockedUsersShow() {
    api.getBlocked().then(bannedUsers => {
      this.setState({ bannedUsers, isBlockedVisible: true });
    });
  }

  onBlockedUsersHide() {
    const { wasSomeoneUnblocked, sort } = this.state;

    // if someone was unblocked let's reload comments
    if (wasSomeoneUnblocked) {
      api.find({ sort, url }).then(({ comments } = {}) => store.set('comments', comments));
    }

    this.setState({
      isBlockedVisible: false,
      wasSomeoneUnblocked: false,
    });
  }

  onReplyClick(cb) {
    if (this.onPrevReplyClickCallback) {
      this.onPrevReplyClickCallback();
    }

    this.onPrevReplyClickCallback = cb;
  }

  onSortChange(sort) {
    if (sort === this.state.sort) return;

    this.setState({ sort, isCommentsListLoading: true });

    try {
      localStorage.setItem(LS_SORT_KEY, sort);
    } catch (e) {
      // can't save; ignore it
    }

    api.find({ sort, url })
      .then(({ comments } = {}) => store.set('comments', comments))
      .finally(() => {
        this.setState({ isCommentsListLoading: false });
      });
  }

  onUnblockSomeone() {
    this.setState({ wasSomeoneUnblocked: true });
  }

  addComment(data) {
    store.addComment(data);
    this.setState({ comments: store.get('comments') });

    api.getComment({ id: data.id }).then(comment => {
      store.replaceComment(comment);
      this.setState({ comments: store.get('comments') });
      store.set('replyingCommentId', null);
    });
  }

  render({}, { config = {}, comments = [], user, sort, isLoaded, isBlockedVisible, isCommentsListLoading, bannedUsers, replyingCommentId }) {
    if (!isLoaded) {
      return (
        <div id={NODE_ID}>
          <div className="root">
            <Preloader mix="root__preloader"/>
          </div>
        </div>
      );
    }

    // TODO: i think we should do it on backend
    const pinnedComments = store.getPinnedComments();
    const isGuest = !Object.keys(user).length;

    return (
      <div id={NODE_ID}>
        <div className="root">
          <AuthPanel
            user={user}
            sort={sort}
            providers={config.auth_providers}
            onSignIn={this.onSignIn}
            onSignOut={this.onSignOut}
            onBlockedUsersShow={this.onBlockedUsersShow}
            onBlockedUsersHide={this.onBlockedUsersHide}
            onSortChange={this.onSortChange}
          />

          {
            !isBlockedVisible && (
              <div className="root__main">
                {
                  !isGuest && (
                    <Input
                      mix="root__input"
                      onSubmit={this.addComment}
                    />
                  )
                }

                {
                  !!pinnedComments.length && (
                    <div className="root__pinned-comments">
                      {
                        pinnedComments.map(comment => (
                          <Comment
                            data={comment}
                            mods={{ level: 0, disabled: true }}
                            mix="root__pinned-comment"
                          />
                        ))
                      }
                    </div>
                  )
                }

                {
                  !!comments.length && !isCommentsListLoading && (
                    <div className="root__threads">
                      {
                        comments.map(thread => (
                          <Thread
                            mix="root__thread"
                            mods={{ level: 0 }}
                            data={thread}
                            onReply={this.addComment}
                            onReplyClick={this.onReplyClick}
                          />
                        ))
                      }
                    </div>
                  )
                }

                {
                  isCommentsListLoading && (
                    <div className="root__threads">
                      <Preloader mix="root__preloader"/>
                    </div>
                  )
                }
              </div>
            )
          }

          {
            isBlockedVisible && (
              <div className="root__main">
                <BlockedUsers users={bannedUsers} onUnblock={this.onUnblockSomeone}/>
              </div>
            )
          }

          <p className="root__copyright">
            Powered by <a href="https://remark42.com/" className="root__copyright-link">Remark42</a>
          </p>
        </div>
      </div>
    );
  }
}
