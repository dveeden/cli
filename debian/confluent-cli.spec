Name: confluent-cli
Version: %{__version}
Release: %{__release}%{?dist}

Summary: CLI for Confluent Cloud and Confluent Platform
Group: Applications/Internet
License: Confluent License Agreement
Source0: confluent-cli-%{version}.tar.gz
URL: http://confluent.io
BuildRoot: %{_tmppath}/%{name}-%{version}-root
Vendor: Confluent, Inc.
Packager: Confluent Packaging <packages@confluent.io>

%description
The Confluent CLI helps you manage your Confluent Cloud and Confluent Platform deployments, right from the terminal.

%define __jar_repack %{nil}
%define _binaries_in_noarch_packages_terminate_build 0

%pre

%post

%preun

%postun

%prep

%setup

%build

%install
# Clean out any previous builds not on slash
[ "%{buildroot}" != "/" ] && %{__rm} -rf %{buildroot}
%{__mkdir_p} %{buildroot}
%{__cp} -R * %{buildroot}

%files
%defattr(-,root,root)
/usr/bin/*
/usr/libexec/cli/
%doc
/usr/share/doc/cli/

%clean
#used to cleanup things outside the build area and possibly inside.
[ "%{buildroot}" != "/" ] && %{__rm} -rf %{buildroot}

%changelog
* Fri Jul 24 2020 Confluent Packaging <packages@confluent.io>
- Initial import
