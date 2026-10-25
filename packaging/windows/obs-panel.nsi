; Van Link OBS Panel installer: adds the Van Link monitoring panel inside OBS Studio.
Unicode true
!include "MUI2.nsh"
!include "LogicLib.nsh"

!ifndef VERSION
  !define VERSION "0.1.2"
!endif
!define APP "Van Link OBS Panel"
!define COMPANY "Gaz Media Productions"
!define HELPER "vanlink-obs-panel.exe"
!define UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\VanLinkOBSPanel"

Name "${APP}"
OutFile "..\..\dist\VanLink-OBS-Panel-Setup-${VERSION}.exe"
InstallDir "$LOCALAPPDATA\Van Link OBS Panel"
RequestExecutionLevel user
SetCompressor /SOLID lzma
BrandingText "${APP} ${VERSION} - ${COMPANY}"

VIProductVersion "${VERSION}.0"
VIAddVersionKey "ProductName" "${APP}"
VIAddVersionKey "FileDescription" "${APP} installer"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "ProductVersion" "${VERSION}"
VIAddVersionKey "CompanyName" "${COMPANY}"
VIAddVersionKey "LegalCopyright" "(c) 2026 ${COMPANY}"

!define MUI_ICON "vanlink.ico"
!define MUI_UNICON "vanlink.ico"
!define MUI_WELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_UNWELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_WELCOMEPAGE_TITLE "Van Link panel for OBS"
!define MUI_WELCOMEPAGE_TEXT "This adds the Van Link monitoring panel inside OBS Studio: live status, signal, delay and the GO LIVE controls, right next to your OBS preview.$\r$\n$\r$\nPlease close OBS before continuing.$\r$\n$\r$\nVan Link itself must also be installed on this computer."
!define MUI_FINISHPAGE_TITLE "The Van Link panel is in OBS"
!define MUI_FINISHPAGE_TEXT "Open OBS. The Van Link panel appears as a dock.$\r$\n$\r$\nIf you don't see it, click View > Docks > Van Link in OBS.$\r$\n$\r$\nThe panel shows information while the Van Link app is running."

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File "..\..\dist\${HELPER}"
  File "vanlink.ico"

  retry:
  nsExec::ExecToStack '"$INSTDIR\${HELPER}" install'
  Pop $0
  Pop $1
  ${If} $0 == "2"
    MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "OBS is open.$\r$\n$\r$\nPlease close OBS completely, then click Retry." IDRETRY retry
    Abort "Cancelled: OBS was still open."
  ${ElseIf} $0 == "3"
    MessageBox MB_OK|MB_ICONEXCLAMATION "OBS has not been set up on this computer yet.$\r$\n$\r$\nOpen OBS once, close it, then run this installer again."
    Abort "OBS not found."
  ${ElseIf} $0 != "0"
    MessageBox MB_OK|MB_ICONSTOP "The panel could not be added:$\r$\n$1"
    Abort "Failed."
  ${EndIf}

  WriteUninstaller "$INSTDIR\Uninstall.exe"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayName" "${APP}"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayVersion" "${VERSION}"
  WriteRegStr HKCU "${UNINST_KEY}" "Publisher" "${COMPANY}"
  WriteRegStr HKCU "${UNINST_KEY}" "DisplayIcon" "$INSTDIR\vanlink.ico"
  WriteRegStr HKCU "${UNINST_KEY}" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKCU "${UNINST_KEY}" "NoRepair" 1
SectionEnd

Section "Uninstall"
  unretry:
  nsExec::ExecToStack '"$INSTDIR\${HELPER}" remove'
  Pop $0
  Pop $1
  ${If} $0 == "2"
    MessageBox MB_RETRYCANCEL|MB_ICONEXCLAMATION "OBS is open.$\r$\n$\r$\nPlease close OBS, then click Retry." IDRETRY unretry
    Abort
  ${EndIf}
  Delete "$INSTDIR\${HELPER}"
  Delete "$INSTDIR\vanlink.ico"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "${UNINST_KEY}"
SectionEnd
