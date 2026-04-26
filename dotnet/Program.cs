using System.Reflection;
using System.Runtime.InteropServices;

var version = typeof(Program).Assembly.GetName().Version?.ToString(3) ?? "dev";

var binName = RuntimeInformation.IsOSPlatform(OSPlatform.Windows)
    ? "clang-format-swig.exe"
    : "clang-format-swig";

var binary = ExtractEmbeddedBinary();

var psi = new ProcessStartInfo(binary) { UseShellExecute = false };
foreach (var arg in args)
  psi.ArgumentList.Add(arg);

using var proc = Process.Start(psi) ?? throw new InvalidOperationException("Failed to start clang-format-swig");
await proc.WaitForExitAsync();
return proc.ExitCode;

// ── helpers ──────────────────────────────────────────────────────────────────

string GetCacheDir()
{
  var root = RuntimeInformation.IsOSPlatform(OSPlatform.Windows)
      ? Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData)
      : Environment.GetEnvironmentVariable("XDG_CACHE_HOME")
        ?? Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.UserProfile), ".cache");
  return Path.Combine(root, "clang-format-swig", version);
}

string GetPlatformKey()
{
  string os =
      RuntimeInformation.IsOSPlatform(OSPlatform.Linux) ? "linux" :
      RuntimeInformation.IsOSPlatform(OSPlatform.OSX) ? "darwin" :
      RuntimeInformation.IsOSPlatform(OSPlatform.Windows) ? "windows" :
      throw new PlatformNotSupportedException(RuntimeInformation.OSDescription);
  string arch = RuntimeInformation.ProcessArchitecture switch
  {
    Architecture.X64 => "amd64",
    Architecture.Arm64 => "arm64",
    var a => throw new PlatformNotSupportedException(a.ToString()),
  };
  return $"{os}_{arch}";
}

string ExtractEmbeddedBinary()
{
  var key = GetPlatformKey();
  var assembly = typeof(Program).Assembly;
  // Resource names look like: clang_format_swig.native.linux_amd64.clang-format-swig
  var resourceName = assembly.GetManifestResourceNames()
      .FirstOrDefault(n => n.Contains(key))
      ?? throw new PlatformNotSupportedException($"No embedded binary for platform {key}.");

  var cacheDir = GetCacheDir();
  var binPath = Path.Combine(cacheDir, binName);

  if (!File.Exists(binPath))
  {
    Directory.CreateDirectory(cacheDir);
    using var stream = assembly.GetManifestResourceStream(resourceName)!;
    using var file = File.Create(binPath);
    stream.CopyTo(file);

    if (!RuntimeInformation.IsOSPlatform(OSPlatform.Windows))
    {
      File.SetUnixFileMode(
        binPath,
        UnixFileMode.UserRead
        | UnixFileMode.UserWrite
        | UnixFileMode.UserExecute
        | UnixFileMode.GroupRead
        | UnixFileMode.GroupExecute
        | UnixFileMode.OtherRead
        | UnixFileMode.OtherExecute);
    }
  }

  return binPath;
}
