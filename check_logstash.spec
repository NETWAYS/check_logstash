# use the following to build a specific version
# obtain hash with  `git ls-remote https://github.com/NETWAYS/check_logstash.git`

%global commit0 3852b12dbc98b94b17e4e45993c6776af84234d2
%global shortcommit0 %(c=%{commit0}; echo ${c:0:7})

%global debug_package %{nil}

Summary: check_logstash - a monitoring plugin to connect to the Logstash API
Name: check_logstash
Version: 0.6.1
Release: 0
Group: Operating System
Vendor: NETways GmbH
Packager: NETways GmbH
License: GPL 3+
# use the following for a specific commit
Source0:  https://github.com/NETWAYS/%{name}/archive/%{commit0}.tar.gz#/%{name}-%{shortcommit0}.tar.gz
# use the following for a specific tag
# Source0:  https://github.com/NETWAYS/%{name}/archive/v%{version}.tar.gz#/%{name}-%{version}.tar.gz
#BuildRequires: rubygem-rake,rubygem-rspec
Requires: ruby
Provides: check_logstash

%description
check_logstash - a monitoring plugin to connect to the Logstash API

%prep

# ATTENTION: remember to use double %% when commenting macros!

# use the following for a specific commit
%autosetup -n %{name}-%{commit0}

# use the following for a specific tag
# %%autosetup -n %{name}-%{version}

%build
# rake

%install
mkdir -p $RPM_BUILD_ROOT/usr/lib64/nagios/plugins
cp -r check_logstash $RPM_BUILD_ROOT/usr/lib64/nagios/plugins

%files
/usr/lib64/nagios/plugins/check_logstash

%changelog

