/** @jsx h */
import { Component, h, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { Button } from '@app/components/button';
import { Theme } from '@app/common/types';

interface Props {
  title: string;
  titleClass?: string;
  heading?: string;
  isActive?: boolean;
  onTitleClick?: () => void;
  mix?: string;
  theme: Theme;
  onOpen?: (root: HTMLDivElement) => unknown;
  onClose?: (root: HTMLDivElement) => unknown;
}

interface State {
  isActive: boolean;
}

export default class Dropdown extends Component<Props, State> {
  rootNode?: HTMLDivElement;

  constructor(props: Props) {
    super(props);

    this.state = {
      isActive: props.isActive || false,
    };

    this.onOutsideClick = this.onOutsideClick.bind(this);
    this.receiveMessage = this.receiveMessage.bind(this);
  }

  onTitleClick() {
    const isActive = !this.state.isActive;
    this.setState(
      {
        isActive,
      },
      () => {
        if (isActive && this.props.onOpen) {
          this.props.onOpen(this.rootNode!);
        }
        if (!isActive && this.props.onClose) {
          this.props.onClose(this.rootNode!);
        }

        if (this.props.onTitleClick) {
          this.props.onTitleClick();
        }
      }
    );
  }

  receiveMessage(e: { data: string | object }) {
    try {
      const data = typeof e.data === 'string' ? JSON.parse(e.data) : e.data;

      if (!data.clickOutside) return;
      if (!this.state.isActive) return;
      this.setState(
        {
          isActive: false,
        },
        () => {
          this.props.onClose && this.props.onClose(this.rootNode!);
        }
      );
    } catch (e) {}
  }

  onOutsideClick(e: MouseEvent) {
    if (!this.rootNode || this.rootNode.contains(e.target as Node) || !this.state.isActive) return;
    this.setState(
      {
        isActive: false,
      },
      () => {
        this.props.onClose && this.props.onClose(this.rootNode!);
      }
    );
  }

  componentDidMount() {
    document.addEventListener('click', this.onOutsideClick);

    window.addEventListener('message', this.receiveMessage);
  }

  componentWillUnmount() {
    document.removeEventListener('click', this.onOutsideClick);

    window.removeEventListener('message', this.receiveMessage);
  }

  render(props: RenderableProps<Props>, { isActive }: State) {
    const { title, titleClass, heading, children, mix } = props;

    return (
      <div className={b('dropdown', { mix }, { theme: props.theme, active: isActive })} ref={r => (this.rootNode = r)}>
        <Button
          aria-haspopup="listbox"
          aria-expanded={isActive && 'true'}
          mix="dropdown__title"
          type="button"
          onClick={() => this.onTitleClick()}
          theme="light"
          className={titleClass}
        >
          {title}
        </Button>

        <div className="dropdown__content" tabIndex={-1} role="listbox">
          {heading && <div className="dropdown__heading">{heading}</div>}
          <div className="dropdown__items">{children}</div>
        </div>
      </div>
    );
  }
}
