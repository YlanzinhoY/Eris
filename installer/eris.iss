#ifndef MyAppVersion
  #define MyAppVersion "0.0.0"
#endif

#ifndef MySourceDir
  #define MySourceDir "..\dist\eris-dev-windows-amd64"
#endif

[Setup]
AppId={{C6102529-514D-4428-B2D4-60F961E8C1A4}
AppName=Éris
AppVersion={#MyAppVersion}
AppVerName=Éris {#MyAppVersion}
AppPublisher=YlanzinhoY
AppPublisherURL=https://github.com/YlanzinhoY/Eris
AppSupportURL=https://github.com/YlanzinhoY/Eris/issues
AppUpdatesURL=https://github.com/YlanzinhoY/Eris/releases
DefaultDirName={localappdata}\Programs\Eris
DisableProgramGroupPage=yes
PrivilegesRequired=lowest
OutputDir=..\dist
OutputBaseFilename=eris-windows-amd64-setup
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
ChangesEnvironment=yes
CloseApplications=yes
RestartApplications=no
SetupLogging=yes
UninstallDisplayIcon={app}\eris.exe
UninstallDisplayName=Éris {#MyAppVersion}
VersionInfoCompany=YlanzinhoY
VersionInfoDescription=Instalador da CLI Éris
VersionInfoProductName=Éris
VersionInfoProductVersion={#MyAppVersion}
VersionInfoVersion={#MyAppVersion}

[Languages]
Name: "brazilianportuguese"; MessagesFile: "compiler:Languages\BrazilianPortuguese.isl"

[Files]
Source: "{#MySourceDir}\eris.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#MySourceDir}\games.json"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#MySourceDir}\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "{#MySourceDir}\CHANGELOG.md"; DestDir: "{app}"; Flags: ignoreversion

[Code]
const
  PathStateKey = 'Software\YlanzinhoY\Eris';

function NormalizePathEntry(Value: String): String;
begin
  Result := Trim(Value);

  if (Length(Result) >= 2) and
     (Result[1] = '"') and
     (Result[Length(Result)] = '"') then
  begin
    Delete(Result, Length(Result), 1);
    Delete(Result, 1, 1);
  end;

  while (Length(Result) > 3) and (Result[Length(Result)] = '\') do
    Delete(Result, Length(Result), 1);
end;

function AddPathEntry(PathValue, Entry: String): String;
var
  Entries: TArrayOfString;
  Index: Integer;
  NormalizedEntry: String;
begin
  NormalizedEntry := NormalizePathEntry(Entry);
  Entries := StringSplit(PathValue, [';'], stExcludeEmpty);

  for Index := 0 to GetArrayLength(Entries) - 1 do
  begin
    if CompareText(NormalizePathEntry(Entries[Index]), NormalizedEntry) = 0 then
    begin
      Result := PathValue;
      Exit;
    end;
  end;

  if Trim(PathValue) = '' then
    Result := Entry
  else if PathValue[Length(PathValue)] = ';' then
    Result := PathValue + Entry
  else
    Result := PathValue + ';' + Entry;
end;

function RemovePathEntry(PathValue, Entry: String): String;
var
  SegmentStart: Integer;
  RelativeSeparator: Integer;
  SegmentEnd: Integer;
  SegmentValue: String;
  NormalizedEntry: String;
begin
  Result := PathValue;
  SegmentStart := 1;
  NormalizedEntry := NormalizePathEntry(Entry);

  while SegmentStart <= Length(PathValue) + 1 do
  begin
    RelativeSeparator := Pos(';', Copy(PathValue, SegmentStart, Length(PathValue)));
    if RelativeSeparator = 0 then
      SegmentEnd := Length(PathValue) + 1
    else
      SegmentEnd := SegmentStart + RelativeSeparator - 1;

    SegmentValue := Copy(PathValue, SegmentStart, SegmentEnd - SegmentStart);
    if CompareText(NormalizePathEntry(SegmentValue), NormalizedEntry) = 0 then
    begin
      if SegmentStart = 1 then
      begin
        if SegmentEnd <= Length(PathValue) then
          Result := Copy(PathValue, SegmentEnd + 1, Length(PathValue))
        else
          Result := '';
      end
      else
        Result := Copy(PathValue, 1, SegmentStart - 2) +
          Copy(PathValue, SegmentEnd, Length(PathValue));

      Exit;
    end;

    if SegmentEnd > Length(PathValue) then
      Exit;

    SegmentStart := SegmentEnd + 1;
  end;
end;

procedure AddApplicationToPath;
var
  CurrentPath: String;
  UpdatedPath: String;
  ApplicationPath: String;
begin
  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', CurrentPath) then
    CurrentPath := '';

  ApplicationPath := ExpandConstant('{app}');
  UpdatedPath := AddPathEntry(CurrentPath, ApplicationPath);

  if CompareText(CurrentPath, UpdatedPath) <> 0 then
  begin
    if not RegWriteDWordValue(
      HKEY_CURRENT_USER, PathStateKey, 'PathEntryAdded', 1) then
      RaiseException('Não foi possível registrar a instalação do Éris.');

    if not RegWriteExpandStringValue(
      HKEY_CURRENT_USER, 'Environment', 'Path', UpdatedPath) then
    begin
      RegDeleteValue(HKEY_CURRENT_USER, PathStateKey, 'PathEntryAdded');
      RaiseException('Não foi possível adicionar o Éris ao PATH do usuário.');
    end;

    Log(Format('Adicionado ao PATH do usuário: %s', [ApplicationPath]));
  end;
end;

procedure RemoveApplicationFromPath;
var
  CurrentPath: String;
  UpdatedPath: String;
  ApplicationPath: String;
  PathEntryAdded: Cardinal;
begin
  if not RegQueryDWordValue(
    HKEY_CURRENT_USER, PathStateKey, 'PathEntryAdded', PathEntryAdded) or
    (PathEntryAdded <> 1) then
    Exit;

  if not RegQueryStringValue(HKEY_CURRENT_USER, 'Environment', 'Path', CurrentPath) then
    Exit;

  ApplicationPath := ExpandConstant('{app}');
  UpdatedPath := RemovePathEntry(CurrentPath, ApplicationPath);

  if CompareText(CurrentPath, UpdatedPath) <> 0 then
  begin
    if UpdatedPath = '' then
    begin
      if not RegDeleteValue(HKEY_CURRENT_USER, 'Environment', 'Path') then
        RaiseException('Não foi possível remover o Éris do PATH do usuário.');
    end
    else if not RegWriteExpandStringValue(
      HKEY_CURRENT_USER, 'Environment', 'Path', UpdatedPath) then
      RaiseException('Não foi possível remover o Éris do PATH do usuário.');

    Log(Format('Removido do PATH do usuário: %s', [ApplicationPath]));
  end;

  RegDeleteKeyIncludingSubkeys(HKEY_CURRENT_USER, PathStateKey);
end;

procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
    AddApplicationToPath;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
begin
  if CurUninstallStep = usUninstall then
    RemoveApplicationFromPath;
end;
