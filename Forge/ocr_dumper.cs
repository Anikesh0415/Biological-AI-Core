using System;
using System.Drawing;
using System.Drawing.Imaging;
using System.IO;
using System.Threading.Tasks;
using System.Windows.Forms;
using Windows.Graphics.Imaging;
using Windows.Media.Ocr;
using Windows.Storage;

namespace OCRDumper
{
    class Program
    {
        static void Main(string[] args)
        {
            string tempImage = "temp_ocr.bmp";
            
            try {
                // 1. Capture screen to Bitmap
                var bounds = Screen.PrimaryScreen.Bounds;
                using (var bitmap = new Bitmap(bounds.Width, bounds.Height))
                {
                    using (var g = Graphics.FromImage(bitmap))
                    {
                        g.CopyFromScreen(Point.Empty, Point.Empty, bounds.Size);
                    }
                    bitmap.Save(tempImage, ImageFormat.Bmp);
                }

                // 2. Load into WinRT SoftwareBitmap
                var fileTask = StorageFile.GetFileFromPathAsync(Path.GetFullPath(tempImage)).AsTask();
                fileTask.Wait();
                StorageFile file = fileTask.Result;

                var streamTask = file.OpenAsync(FileAccessMode.Read).AsTask();
                streamTask.Wait();
                using (var stream = streamTask.Result)
                {
                    var decoderTask = BitmapDecoder.CreateAsync(stream).AsTask();
                    decoderTask.Wait();
                    var decoder = decoderTask.Result;

                    var softwareBitmapTask = decoder.GetSoftwareBitmapAsync().AsTask();
                    softwareBitmapTask.Wait();
                    var softwareBitmap = softwareBitmapTask.Result;

                    // 3. Run OCR
                    var engine = OcrEngine.TryCreateFromLanguage(OcrEngine.AvailableRecognizerLanguages[0]);
                    var resultTask = engine.RecognizeAsync(softwareBitmap).AsTask();
                    resultTask.Wait();
                    var result = resultTask.Result;

                    // 4. Output JSON
                    Console.WriteLine("[");
                    bool first = true;
                    foreach (var line in result.Lines)
                    {
                        foreach (var word in line.Words)
                        {
                            if (!first) Console.WriteLine(",");
                            first = false;
                            
                            var rect = word.BoundingRect;
                            string text = word.Text.Replace("\"", "\\\"").Replace("\n", "").Replace("\r", "");
                            Console.Write(string.Format("  {{\"text\": \"{0}\", \"x\": {1}, \"y\": {2}, \"width\": {3}, \"height\": {4}}}", text, rect.X, rect.Y, rect.Width, rect.Height));
                        }
                    }
                    Console.WriteLine("\n]");
                }
            } 
            catch (Exception ex) {
                Console.Error.WriteLine(ex.Message);
            }
            finally {
                if (File.Exists(tempImage)) {
                    File.Delete(tempImage);
                }
            }
        }
    }
}
