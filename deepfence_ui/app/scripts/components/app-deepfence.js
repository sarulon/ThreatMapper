/* eslint-disable no-nested-ternary */
import React from 'react';
import { connect } from 'react-redux';
import { Redirect, Route, HashRouter, Switch } from 'react-router-dom';

// Custom components
import TableJSONViewModal from './common/table-json-view-modal/table-json-view-modal';
import DFModal from './common/modal/index';
import EULAView from './common/eula-view/eula-view';

import TopologyView from './topology-view/topology-view';
import VulnerabilityView from './vulnerability-view/index';
import CVEDetailsView from './vulnerability-view/vulnerability-view';
import NotificationsView from './notification-view/notification-view';
import SettingsView from './settings-view/settings-view';
import RegistryVulnerabilityScan from './vulnerability-view/registry-scan/index';

import LoginView from './auth-module/login-view/login-view';
import RegisterView from './auth-module/register-view/register-view';
import ForgotPasswordView from './auth-module/forgot-password-view/forgot-password-view';
import ResetPasswordView from './auth-module/reset-password-view/reset-password-view';
import RegisterViaInviteView from './auth-module/register-via-invite-view/register-via-invite-view';
import { isUserSessionActive } from '../helpers/auth-helper';

const PrivateRoute = ({ component: Component, ...rest }) => (
  <Route
    {...rest}
    render={props =>
      isUserSessionActive() ? (
        <Component {...props} />
      ) : (
        <Redirect to="/login" />
      )
    }
  />
);

class DeepFenceApp extends React.Component {
  constructor() {
    super();
    this.state = {};
  }

  render() {
    const { isTableJSONViewModal } = this.props;
    return (
      <div className="dashboard-wrapper">
        <HashRouter>
          <Switch>
            <Route
              path="/login"
              render={() =>
                isUserSessionActive() ? (
                  <Redirect to="/topology" />
                ) : (
                  <LoginView />
                )
              }
            />
            <Route
              path="/register"
              render={() =>
                isUserSessionActive() ? (
                  <Redirect to="/topology" />
                ) : (
                  <RegisterView />
                )
              }
            />
            <Route
              path="/forgot-password"
              render={() =>
                isUserSessionActive() ? (
                  <Redirect to="/topology" />
                ) : (
                  <ForgotPasswordView />
                )
              }
            />
            <Route
              path="/password-reset"
              render={() =>
                isUserSessionActive() ? (
                  <Redirect to="/topology" />
                ) : (
                  <ResetPasswordView />
                )
              }
            />
            <Route
              path="/invite-accept"
              render={() =>
                isUserSessionActive() ? (
                  <Redirect to="/topology" />
                ) : (
                  <RegisterViaInviteView />
                )
              }
            />
            <Route
              path="/user-agreement"
              target="_blank"
              render={(props) => <EULAView {...props} />}
            />

            <PrivateRoute path="/topology" component={TopologyView} />
            <PrivateRoute
              path="/vulnerability/details/:scanId"
              component={CVEDetailsView}
            />
            <PrivateRoute path="/vulnerability" component={VulnerabilityView} />
            <PrivateRoute
              path="/registry_vulnerability_scan"
              component={RegistryVulnerabilityScan}
            />
            <PrivateRoute path="/notification" component={NotificationsView} />
            <PrivateRoute path="/settings" component={SettingsView} />

            <Route
              path="*"
              render={() =>
                isUserSessionActive() ? (
                  <Redirect to="/topology" />
                ) : (
                  <Redirect to="/login" />
                )
              }
            />
          </Switch>
        </HashRouter>

        {isTableJSONViewModal && <TableJSONViewModal eventTypes={['click']} />}
        <DFModal />
      </div>
    );
  }
}

function mapStateToProps(state) {
  return {
    isTableJSONViewModal: state.get('isTableJSONViewModal'),
  };
}

export default connect(mapStateToProps)(DeepFenceApp);
