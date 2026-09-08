; Installateur Windows de l'agent Festival Command Center (NSIS).
; Construit par GoReleaser (hook) : makensis -DVERSION=x.y.z -DAGENT=chemin\agent.exe -DTRAY=chemin\agent-tray.exe -DOUT=chemin\Setup.exe festival-agent.nsi
Unicode true
!include "MUI2.nsh"
!include "x64.nsh"

!ifndef VERSION
  !define VERSION "0.0.0"
!endif
Name "Festival Command Center Agent ${VERSION}"
OutFile "${OUT}"
InstallDir "$PROGRAMFILES64\Festival Command Center"
InstallDirRegKey HKLM "Software\FestivalCommandCenter" "InstallDir"
RequestExecutionLevel admin
SetCompressor /SOLID lzma

!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN
!define MUI_FINISHPAGE_RUN_TEXT "Ouvrir l'icône et la fenêtre de configuration"
!define MUI_FINISHPAGE_RUN_FUNCTION LaunchTray
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "French"

Section "Agent" SecAgent
  SectionIn RO
  SetOutPath "$INSTDIR"
  ; arrêter ce qui tourne déjà (mise à jour de l'installation)
  nsExec::ExecToLog 'taskkill /F /IM agent-tray.exe'
  nsExec::ExecToLog '"$INSTDIR\agent.exe" -config "$INSTDIR\agent.toml" stop'
  File "/oname=agent.exe" "${AGENT}"
  File "/oname=agent-tray.exe" "${TRAY}"
  WriteRegStr HKLM "Software\FestivalCommandCenter" "InstallDir" "$INSTDIR"
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  ; service Windows (agent.exe crée agent.toml s'il manque)
  nsExec::ExecToLog '"$INSTDIR\agent.exe" -config "$INSTDIR\agent.toml" install'
  ; icône au démarrage de session, pour tous les utilisateurs
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Run" "FestivalCommandCenterTray" '"$INSTDIR\agent-tray.exe"'
  ; menu Démarrer
  CreateDirectory "$SMPROGRAMS\Festival Command Center"
  CreateShortcut "$SMPROGRAMS\Festival Command Center\Agent Festival Command Center.lnk" "$INSTDIR\agent-tray.exe"
  CreateShortcut "$SMPROGRAMS\Festival Command Center\Désinstaller.lnk" "$INSTDIR\Uninstall.exe"
  ; Ajout/Suppression de programmes
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FestivalCommandCenterAgent" "DisplayName" "Festival Command Center Agent"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FestivalCommandCenterAgent" "DisplayVersion" "${VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FestivalCommandCenterAgent" "Publisher" "Festival Command Center"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FestivalCommandCenterAgent" "UninstallString" '"$INSTDIR\Uninstall.exe"'
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FestivalCommandCenterAgent" "DisplayIcon" "$INSTDIR\agent-tray.exe"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FestivalCommandCenterAgent" "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FestivalCommandCenterAgent" "NoRepair" 1
SectionEnd

Function LaunchTray
  ; lancer l'icône sans élévation (via explorer), puis la fenêtre de configuration
  Exec '"$WINDIR\explorer.exe" "$INSTDIR\agent-tray.exe"'
  Sleep 1500
  Exec '"$WINDIR\explorer.exe" "$INSTDIR\agent-tray.exe" --window'
FunctionEnd

Section "Uninstall"
  nsExec::ExecToLog 'taskkill /F /IM agent-tray.exe'
  nsExec::ExecToLog '"$INSTDIR\agent.exe" -config "$INSTDIR\agent.toml" uninstall'
  DeleteRegValue HKLM "Software\Microsoft\Windows\CurrentVersion\Run" "FestivalCommandCenterTray"
  DeleteRegValue HKCU "Software\Microsoft\Windows\CurrentVersion\Run" "FestivalCommandCenterTray"
  Delete "$INSTDIR\agent.exe"
  Delete "$INSTDIR\agent.exe.old"
  Delete "$INSTDIR\agent-tray.exe"
  Delete "$INSTDIR\agent-tray.exe.old"
  Delete "$INSTDIR\install.log"
  Delete "$INSTDIR\Uninstall.exe"
  MessageBox MB_YESNO "Supprimer aussi la configuration (agent.toml) et l'inscription au réseau (dossier state) ?" IDNO keep
    Delete "$INSTDIR\agent.toml"
    RMDir /r "$INSTDIR\state"
  keep:
  RMDir "$INSTDIR"
  Delete "$SMPROGRAMS\Festival Command Center\*.lnk"
  RMDir "$SMPROGRAMS\Festival Command Center"
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\FestivalCommandCenterAgent"
  DeleteRegKey HKLM "Software\FestivalCommandCenter"
SectionEnd
