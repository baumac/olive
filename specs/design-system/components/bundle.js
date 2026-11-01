/* @ds-bundle: {"format":4,"namespace":"Sponga47","components":[{"name":"Button"},{"name":"TextField"},{"name":"Select"},{"name":"Checkbox"},{"name":"Card"},{"name":"Tabs"},{"name":"Badge"},{"name":"Alert"}]} */
(function () {
  var React = window.React;
  var h = React.createElement;

  function cx() {
    return Array.prototype.filter.call(arguments, Boolean).join(' ');
  }

  function Button(props) {
    var { variant = 'secondary', size = 'md', type = 'button', className, children, ...rest } = props;
    return h('button', {
      ...rest,
      type: type,
      className: cx('sp-btn', 'sp-btn-' + variant, size === 'sm' && 'sp-btn-sm', className)
    }, children);
  }

  function FieldShell(p) {
    var hintId = p.id + '-hint';
    var message = p.error || p.hint;
    return h('div', { className: cx('sp-field', p.error && 'sp-field-invalid', p.className) },
      p.label && h('label', { className: 'sp-label', htmlFor: p.id }, p.label),
      p.control(message ? hintId : undefined),
      message && h('p', { id: hintId, className: p.error ? 'sp-error' : 'sp-hint' }, message)
    );
  }

  function TextField(props) {
    var { label, hint, error, multiline, className, id, ...rest } = props;
    var auto = React.useId();
    var fieldId = id || auto;
    return h(FieldShell, {
      id: fieldId, label: label, hint: hint, error: error, className: className,
      control: function (describedBy) {
        return h(multiline ? 'textarea' : 'input', {
          ...rest,
          id: fieldId,
          className: cx('sp-input', multiline && 'sp-textarea'),
          'aria-invalid': error ? true : undefined,
          'aria-describedby': describedBy
        });
      }
    });
  }

  function Select(props) {
    var { label, hint, error, options = [], placeholder, className, id, ...rest } = props;
    var auto = React.useId();
    var fieldId = id || auto;
    return h(FieldShell, {
      id: fieldId, label: label, hint: hint, error: error, className: className,
      control: function (describedBy) {
        return h('div', { className: 'sp-select' },
          h('select', {
            ...rest,
            id: fieldId,
            className: 'sp-input',
            'aria-invalid': error ? true : undefined,
            'aria-describedby': describedBy
          },
            placeholder && h('option', { value: '', disabled: true }, placeholder),
            options.map(function (o) {
              return h('option', { key: o.value, value: o.value, disabled: o.disabled }, o.label);
            })
          )
        );
      }
    });
  }

  function Checkbox(props) {
    var { label, hint, className, id, ...rest } = props;
    var auto = React.useId();
    var boxId = id || auto;
    return h('div', { className: cx('sp-check', className) },
      h('input', { ...rest, id: boxId, type: 'checkbox', 'aria-describedby': hint ? boxId + '-hint' : undefined }),
      h('div', null,
        h('label', { htmlFor: boxId }, label),
        hint && h('p', { id: boxId + '-hint', className: 'sp-hint' }, hint)
      )
    );
  }

  function Card(props) {
    var { title, actions, footer, className, children, ...rest } = props;
    return h('section', { ...rest, className: cx('sp-card', className) },
      (title || actions) && h('header', { className: 'sp-card-head' },
        title && h('h3', { className: 'sp-card-title' }, title),
        actions && h('div', { className: 'sp-card-actions' }, actions)
      ),
      h('div', { className: 'sp-card-body' }, children),
      footer && h('footer', { className: 'sp-card-foot' }, footer)
    );
  }

  function Tabs(props) {
    var tabs = props.tabs || [];
    var controlled = props.value !== undefined;
    var state = React.useState(props.defaultValue !== undefined ? props.defaultValue : (tabs[0] && tabs[0].id));
    var current = controlled ? props.value : state[0];
    var base = React.useId();
    var refs = React.useRef({});

    function select(tabId) {
      if (!controlled) state[1](tabId);
      if (props.onChange) props.onChange(tabId);
    }

    function onKeyDown(e, i) {
      var n = tabs.length;
      var next = { ArrowRight: (i + 1) % n, ArrowLeft: (i - 1 + n) % n, Home: 0, End: n - 1 }[e.key];
      if (next === undefined) return;
      e.preventDefault();
      select(tabs[next].id);
      var el = refs.current[tabs[next].id];
      if (el) el.focus();
    }

    var active = tabs.find(function (t) { return t.id === current; });
    return h('div', { className: cx('sp-tabs', props.className) },
      h('div', { role: 'tablist', 'aria-label': props.label, className: 'sp-tablist' },
        tabs.map(function (t, i) {
          var selected = t.id === current;
          return h('button', {
            key: t.id,
            ref: function (el) { refs.current[t.id] = el; },
            type: 'button',
            role: 'tab',
            id: base + '-tab-' + t.id,
            'aria-selected': selected,
            'aria-controls': base + '-panel',
            tabIndex: selected ? 0 : -1,
            className: 'sp-tab',
            onClick: function () { select(t.id); },
            onKeyDown: function (e) { onKeyDown(e, i); }
          }, t.label);
        })
      ),
      h('div', {
        role: 'tabpanel',
        id: base + '-panel',
        'aria-labelledby': base + '-tab-' + current,
        tabIndex: 0,
        className: 'sp-tabpanel'
      }, active ? active.content : null)
    );
  }

  function Badge(props) {
    var { tone = 'neutral', className, children, ...rest } = props;
    return h('span', { ...rest, className: cx('sp-badge', 'sp-badge-' + tone, className) }, children);
  }

  var ICONS = {
    info: 'M10 9v5M10 6.5v.01',
    success: 'M6.5 10.5l2.5 2.5 4.5-5',
    danger: 'M10 6v5M10 13.5v.01'
  };

  function Alert(props) {
    var { tone = 'info', title, className, children, ...rest } = props;
    return h('div', {
      role: tone === 'danger' ? 'alert' : 'status',
      ...rest,
      className: cx('sp-alert', 'sp-alert-' + tone, className)
    },
      h('svg', { className: 'sp-alert-icon', width: 20, height: 20, viewBox: '0 0 20 20', 'aria-hidden': true },
        h('circle', { cx: 10, cy: 10, r: 8 }),
        h('path', { d: ICONS[tone] || ICONS.info })
      ),
      h('div', null,
        title && h('p', { className: 'sp-alert-title' }, title),
        children && h('div', { className: 'sp-alert-body' }, children)
      )
    );
  }

  window.Sponga47 = Object.assign(window.Sponga47 || {}, {
    Button: Button,
    TextField: TextField,
    Select: Select,
    Checkbox: Checkbox,
    Card: Card,
    Tabs: Tabs,
    Badge: Badge,
    Alert: Alert
  });
})();
