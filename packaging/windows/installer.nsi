; Van Link installer for Windows (built with NSIS: makensis -DVERSION=x.y.z installer.nsi)
Unicode true
!include "MUI2.nsh"

!ifndef VERSION
  !define VERSION "0.1.0"
!endif
!define APP "Van Link"
!define COMPANY "Gaz Media Productions"
!define EXE "VanLink.exe"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\VanLink"

Name "${APP}"
OutFile "..\..\dist\VanLink-Setup-${VERSION}.exe"
InstallDir "$PROGRAMFILES64\${APP}"
InstallDirRegKey HKLM "Software\VanLink" "InstallDir"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
BrandingText "${APP} ${VERSION} - ${COMPANY}"

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${APP}"
VIAddVersionKey "FileDescription" "${APP} installer"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "LegalCopyright" "(c) 2026 ${COMPANY}"
VIAddVersionKey "CompanyName" "${COMPANY}"

!define MUI_ICON "vanlink.ico"
!define MUI_UNICON "vanlink.ico"
!define MUI_ABORTWARNING
!define MUI_WELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_WELCOMEPAGE_TITLE "Welcome to ${APP} by ${COMPANY}"
!define MUI_WELCOMEPAGE_TEXT "Van Link sends live video from your outside broadcast van straight to your studio over the internet, with OBS Studio.$\r$\n$\r$\nInstall it on both the van computer and the studio computer. Van Link opens in its own window and also adds a Van Link panel inside OBS.$\r$\n$\r$\nYou will also need OBS Studio 28 or newer.$\r$\n$\r$\nClick Next to continue."
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Open ${APP} now"
!define MUI_FINISHPAGE_RUN_FUNCTION OpenApp
!define MUI_FINISHPAGE_SHOWREADME ""
!define MUI_FINISHPAGE_SHOWREADME_NOTCHECKED
!define MUI_FINISHPAGE_SHOWREADME_TEXT "Show how to turn on OBS remote control"
!define MUI_FINISHPAGE_SHOWREADME_FUNCTION ShowHelp

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Function OpenApp
  ; start without administrator rights, as the person who is logged in
  Exec '"$WINDIR\explorer.exe" "$INSTDIR\${EXE}"'
FunctionEnd

Function ShowHelp
  ExecShell "open" "$INSTDIR\How to connect OBS.txt"
FunctionEnd

Section "Install"
  ; close a running copy first
  nsExec::Exec 'taskkill /IM ${EXE} /F'
  Sleep 500

  SetOutPath "$INSTDIR"
  File "..\..\dist\${EXE}"
  File "vanlink.ico"
  File "How to connect OBS.txt"
  WriteUninstaller "$INSTDIR\Uninstall.exe"

  CreateDirectory "$SMPROGRAMS\${APP}"
  CreateShortcut "$SMPROGRAMS\${APP}\${APP}.lnk" "$INSTDIR\${EXE}" "" "$INSTDIR\vanlink.ico"
  CreateShortcut "$SMPROGRAMS\${APP}\Uninstall ${APP}.lnk" "$INSTDIR\Uninstall.exe"
  CreateShortcut "$DESKTOP\${APP}.lnk" "$INSTDIR\${EXE}" "" "$INSTDIR\vanlink.ico"

  ; let Van Link receive connections through Windows Firewall
  nsExec::Exec 'netsh advfirewall firewall delete rule name="${APP}"'
  nsExec::Exec 'netsh advfirewall firewall add rule name="${APP}" dir=in action=allow program="$INSTDIR\${EXE}" enable=yes profile=any'

  WriteRegStr HKLM "Software\VanLink" "InstallDir" "$INSTDIR"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayName" "${APP}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "${UNINST_KEY}" "Publisher" "${COMPANY}"
  WriteRegStr HKLM "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\vanlink.ico"
  WriteRegStr HKLM "${UNINST_KEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKLM "${UNINST_KEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${UNINST_KEY}" "NoRepair" 1
SectionEnd

Section "Uninstall"
  nsExec::Exec 'taskkill /IM ${EXE} /F'
  Sleep 500
  nsExec::Exec 'netsh advfirewall firewall delete rule name="${APP}"'
  nsExec::Exec 'reg delete HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v VanLink /f'
  Delete "$INSTDIR\${EXE}"
  Delete "$INSTDIR\vanlink.ico"
  Delete "$INSTDIR\How to connect OBS.txt"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\${APP}\${APP}.lnk"
  Delete "$SMPROGRAMS\${APP}\Uninstall ${APP}.lnk"
  RMDir "$SMPROGRAMS\${APP}"
  Delete "$DESKTOP\${APP}.lnk"
  DeleteRegKey HKLM "${UNINST_KEY}"
  DeleteRegKey HKLM "Software\VanLink"
  ; settings in %APPDATA%\VanLink are kept, so pairings survive a reinstall
SectionEnd
