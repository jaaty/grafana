import React, { PureComponent } from 'react';
import CustomScrollbar from 'app/core/components/CustomScrollbar/CustomScrollbar';
import { FadeIn } from 'app/core/components/Animations/FadeIn';

interface Props {
  children: JSX.Element;
  heading: string;
  renderToolbar?: () => JSX.Element;
  toolbarItems?: EditorToolBarView[];
}

export interface EditorToolBarView {
  title: string;
  imgSrc?: string;
  icon?: string;
  disabled?: boolean;
  onClick?: () => void;
  render: (closeFunction: any) => JSX.Element | JSX.Element[];
}

interface State {
  openView?: EditorToolBarView;
  isOpen: boolean;
  fadeIn: boolean;
}

export class EditorTabBody extends PureComponent<Props, State> {
  static defaultProps = {
    toolbarItems: [],
  };

  constructor(props) {
    super(props);

    this.state = {
      openView: null,
      fadeIn: false,
      isOpen: false,
    };
  }

  componentDidMount() {
    this.setState({ fadeIn: true });
  }

  onToggleToolBarView = (item: EditorToolBarView) => {
    this.setState({
      openView: item,
      isOpen: !this.state.isOpen,
    });
  };

  onCloseOpenView = () => {
    this.setState({ isOpen: false });
  };

  static getDerivedStateFromProps(props, state) {
    if (state.openView) {
      const activeToolbarItem = props.toolbarItems.find(
        item => item.title === state.openView.title && item.icon === state.openView.icon
      );
      if (activeToolbarItem) {
        return {
          ...state,
          openView: activeToolbarItem,
        };
      }
    }
    return state;
  }

  renderButton(view: EditorToolBarView) {
    const onClick = () => {
      if (view.onClick) {
        view.onClick();
      }
      this.onToggleToolBarView(view);
    };

    return (
      <div className="nav-buttons" key={view.title + view.icon}>
        <button className="btn navbar-button" onClick={onClick} disabled={view.disabled}>
          {view.icon && <i className={view.icon} />} {view.title}
        </button>
      </div>
    );
  }

  renderOpenView(view: EditorToolBarView) {
    return (
      <div className="toolbar-subview">
        <button className="toolbar-subview__close" onClick={this.onCloseOpenView}>
          <i className="fa fa-chevron-up" />
        </button>
        {view.render(this.onCloseOpenView)}
      </div>
    );
  }

  render() {
    const { children, renderToolbar, heading, toolbarItems } = this.props;
    const { openView, fadeIn, isOpen } = this.state;

    return (
      <>
        <div className="toolbar">
          <div className="toolbar__heading">{heading}</div>
          {renderToolbar && renderToolbar()}
          <div className="gf-form--grow" />
          {toolbarItems.map(item => this.renderButton(item))}
        </div>
        <div className="panel-editor__scroll">
          <CustomScrollbar autoHide={false}>
            <FadeIn in={isOpen} duration={200} unmountOnExit={true}>
              <div className="panel-editor__toolbar-view">{openView && this.renderOpenView(openView)}</div>
            </FadeIn>
            <div className="panel-editor__content">
              <FadeIn in={fadeIn} duration={50}>
                {children}
              </FadeIn>
            </div>
          </CustomScrollbar>
        </div>
      </>
    );
  }
}
