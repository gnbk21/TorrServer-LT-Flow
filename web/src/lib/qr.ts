import QRCode from "qrcode";
export async function generateQrSvg(text: string): Promise<string> {
  try {
    return await QRCode.toString(text, {
      type: "svg",
      margin: 4,
      color: { dark: "#000000", light: "#ffffff" },
    });
  } catch {
    return "";
  }
}
